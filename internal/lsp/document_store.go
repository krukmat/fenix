package lsp // CLSF-41

import "sync"

// Document holds the in-memory state of an open LSP text document.
type Document struct {
	URI     string
	Version int
	Text    string
}

// DocumentStore keeps an in-memory map of open documents keyed by URI. // CLSF-41
// It is safe for concurrent use.
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]Document
}

// NewDocumentStore returns an empty DocumentStore.
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]Document)}
}

// Open registers a document for the given URI, replacing any existing entry.
// Called on textDocument/didOpen.
func (ds *DocumentStore) Open(uri string, version int, text string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.docs[uri] = Document{URI: uri, Version: version, Text: text}
}

// Change updates the text and version of an already-open document.
// If the URI is not known the call is a no-op.
// Called on textDocument/didChange.
func (ds *DocumentStore) Change(uri string, version int, text string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if _, ok := ds.docs[uri]; !ok {
		return
	}
	ds.docs[uri] = Document{URI: uri, Version: version, Text: text}
}

// Close removes a document from the store.
// Called on textDocument/didClose.
func (ds *DocumentStore) Close(uri string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.docs, uri)
}

// Get returns the document for the given URI and whether it was found.
func (ds *DocumentStore) Get(uri string) (Document, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	doc, ok := ds.docs[uri]
	return doc, ok
}
