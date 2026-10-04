package application

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const (
	embeddingDimensions = 256
	embeddingModelName  = "local-feature-hash-v1"
)

type KnowledgeIndexStore interface {
	ReplaceKnowledgeChunks(context.Context, domain.ChunkStrategy, []domain.KnowledgeChunk) error
	KnowledgeIndexStatus(context.Context, string) (domain.KnowledgeIndexStatus, error)
}

type corpusDocument struct{ Source, Title, Content string }

type KnowledgeIndexService struct {
	store KnowledgeIndexStore
	root  string
	now   func() time.Time
}

func NewKnowledgeIndexService(store KnowledgeIndexStore, root string) *KnowledgeIndexService {
	return &KnowledgeIndexService{store: store, root: root, now: time.Now}
}

func (s *KnowledgeIndexService) Status(ctx context.Context) (domain.KnowledgeIndexStatus, error) {
	return s.store.KnowledgeIndexStatus(ctx, s.root)
}

func (s *KnowledgeIndexService) Build(ctx context.Context) (domain.KnowledgeIndexStatus, error) {
	documents, err := loadCorpus(s.root)
	if err != nil {
		return domain.KnowledgeIndexStatus{}, err
	}
	for _, strategy := range []domain.ChunkStrategy{domain.ChunkStrategyFixed, domain.ChunkStrategyStructural} {
		chunks := chunkDocuments(documents, strategy, s.now().UTC())
		for index := range chunks {
			chunks[index].Vector = embedText(chunks[index].Content)
			chunks[index].Dimensions = embeddingDimensions
		}
		if err := s.store.ReplaceKnowledgeChunks(ctx, strategy, chunks); err != nil {
			return domain.KnowledgeIndexStatus{}, err
		}
	}
	return s.store.KnowledgeIndexStatus(ctx, s.root)
}

func loadCorpus(root string) ([]corpusDocument, error) {
	allowed := map[string]bool{".md": true, ".go": true, ".tsx": true, ".ts": true, ".yaml": true, ".yml": true}
	documents := []corpusDocument{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !allowed[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() == 0 || info.Size() > 768<<10 {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := strings.TrimSpace(string(raw))
		if content == "" || !utf8.ValidString(content) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		documents = append(documents, corpusDocument{Source: filepath.ToSlash(rel), Title: filepath.Base(path), Content: content})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan knowledge corpus: %w", err)
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("knowledge corpus is empty at %s", root)
	}
	sort.Slice(documents, func(i, j int) bool { return documents[i].Source < documents[j].Source })
	return documents, nil
}

func chunkDocuments(documents []corpusDocument, strategy domain.ChunkStrategy, createdAt time.Time) []domain.KnowledgeChunk {
	chunks := []domain.KnowledgeChunk{}
	for _, document := range documents {
		var parts []struct{ section, content string }
		if strategy == domain.ChunkStrategyFixed {
			parts = fixedChunks(document.Content, 1400, 220)
		} else {
			parts = structuralChunks(document.Content)
		}
		for index, part := range parts {
			content := strings.TrimSpace(part.content)
			if content == "" {
				continue
			}
			digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%s", strategy, document.Source, index, content)))
			id := fmt.Sprintf("%s-%x", strategy, digest[:6])
			chunks = append(chunks, domain.KnowledgeChunk{ID: id, Strategy: strategy, Source: document.Source, Title: document.Title, Section: part.section, Content: content, CharCount: utf8.RuneCountInString(content), CreatedAt: createdAt})
		}
	}
	return chunks
}

func fixedChunks(content string, size, overlap int) []struct{ section, content string } {
	runes := []rune(content)
	result := []struct{ section, content string }{}
	step := size - overlap
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, struct{ section, content string }{fmt.Sprintf("characters %d–%d", start, end), string(runes[start:end])})
		if end == len(runes) {
			break
		}
	}
	return result
}

var structuralBoundary = regexp.MustCompile(`(?m)^(#{1,4}\s+.+|func\s+\([^)]+\)\s*\w+|func\s+\w+|type\s+\w+|export\s+(?:default\s+)?function\s+\w+)`)

func structuralChunks(content string) []struct{ section, content string } {
	indexes := structuralBoundary.FindAllStringIndex(content, -1)
	if len(indexes) == 0 {
		return packParagraphs("document", content, 1800)
	}
	result := []struct{ section, content string }{}
	if indexes[0][0] > 0 {
		result = append(result, packParagraphs("preamble", content[:indexes[0][0]], 1800)...)
	}
	for index, bounds := range indexes {
		end := len(content)
		if index+1 < len(indexes) {
			end = indexes[index+1][0]
		}
		heading := strings.TrimSpace(content[bounds[0]:bounds[1]])
		result = append(result, packParagraphs(heading, content[bounds[0]:end], 1800)...)
	}
	return result
}

func packParagraphs(section, content string, max int) []struct{ section, content string } {
	paragraphs := strings.Split(content, "\n\n")
	result := []struct{ section, content string }{}
	current := ""
	flush := func() {
		if strings.TrimSpace(current) != "" {
			result = append(result, struct{ section, content string }{section, strings.TrimSpace(current)})
			current = ""
		}
	}
	for _, paragraph := range paragraphs {
		if utf8.RuneCountInString(current)+utf8.RuneCountInString(paragraph)+2 > max {
			flush()
		}
		if utf8.RuneCountInString(paragraph) > max {
			for _, part := range fixedChunks(paragraph, max, 120) {
				result = append(result, struct{ section, content string }{section, part.content})
			}
			continue
		}
		if current != "" {
			current += "\n\n"
		}
		current += paragraph
	}
	flush()
	return result
}

func embedText(text string) []float64 {
	vector := make([]float64, embeddingDimensions)
	tokens := tokenize(text)
	for _, token := range tokens {
		addHashedFeature(vector, "w:"+token, 1)
		runes := []rune(token)
		for n := 3; n <= 4; n++ {
			for start := 0; start+n <= len(runes); start++ {
				addHashedFeature(vector, "g:"+string(runes[start:start+n]), .35)
			}
		}
	}
	norm := 0.0
	for _, value := range vector {
		norm += value * value
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for index := range vector {
			vector[index] /= norm
		}
	}
	return vector
}

func tokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if utf8.RuneCountInString(field) > 1 {
			result = append(result, field)
		}
	}
	return result
}
func addHashedFeature(vector []float64, feature string, weight float64) {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(feature))
	value := hash.Sum64()
	index := int(value % uint64(len(vector)))
	if value&(1<<63) != 0 {
		weight = -weight
	}
	vector[index] += weight
}
