package server

import (
	"github.com/matiasleandrokruk/fenix/internal/domain/relationship"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

func (s *Server) startRelationshipRuntime(bus eventbus.EventBus, chatProvider, embedProvider llm.LLMProvider) {
	repo := relationship.NewSQLiteSignalRepository(s.db)
	summarizer := relationship.NewSummarizer(bus, chatProvider, repo)
	embedder := relationship.NewMemoryEmbedder(bus, embedProvider, repo)
	graph := relationship.NewGraphExtractor(bus, repo)
	trust := relationship.NewTrustDriver(bus, repo)

	s.startBackground(func() { summarizer.Run(s.bgCtx) })
	s.startBackground(func() { embedder.Run(s.bgCtx) })
	s.startBackground(func() { graph.Run(s.bgCtx) })
	s.startBackground(func() { trust.Run(s.bgCtx) })
}
