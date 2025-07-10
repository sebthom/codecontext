package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestRealTreeSitterParsing(t *testing.T) {
	manager := NewManager()

	// Create a test TypeScript file
	testContent := `// Sample TypeScript file
import { Component } from 'react';
import * as fs from 'fs';

interface User {
  id: number;
  name: string;
  email?: string;
}

class UserService {
  private users: User[] = [];

  constructor() {
    this.loadUsers();
  }

  public async getUser(id: number): Promise<User | null> {
    return this.users.find(user => user.id === id) || null;
  }

  public addUser(user: User): void {
    this.users.push(user);
  }

  private loadUsers(): void {
    console.log('Loading users...');
  }
}

export default UserService;
export { User };

function processUsers(users: User[]): void {
  users.forEach(user => {
    console.log(` + "`" + `Processing user: ${user.name}` + "`" + `);
  });
}

const DEFAULT_TIMEOUT = 5000;
let currentUser: User | null = null;`

	// Create temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.ts")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Detect language
	lang := manager.detectLanguage(testFile)
	if lang == nil {
		t.Fatal("Failed to detect TypeScript language")
	}

	if lang.Name != "typescript" {
		t.Errorf("Expected typescript, got %s", lang.Name)
	}

	// Parse the file
	ast, err := manager.ParseFile(testFile, *lang)
	if err != nil {
		t.Fatalf("Failed to parse TypeScript file: %v", err)
	}

	// Verify AST structure
	if ast == nil {
		t.Fatal("AST is nil")
	}

	if ast.Language != "typescript" {
		t.Errorf("Expected language typescript, got %s", ast.Language)
	}

	if ast.Root == nil {
		t.Fatal("AST root is nil")
	}

	if ast.Root.Type != "program" {
		t.Errorf("Expected root type 'program', got %s", ast.Root.Type)
	}

	// Verify we have some children (the AST should have structure)
	if len(ast.Root.Children) == 0 {
		t.Error("AST root has no children - parsing might not be working correctly")
	}

	t.Logf("AST parsed successfully with %d top-level nodes", len(ast.Root.Children))

	// Extract symbols
	symbols, err := manager.ExtractSymbols(ast)
	if err != nil {
		t.Fatalf("Failed to extract symbols: %v", err)
	}

	t.Logf("Extracted %d symbols", len(symbols))

	// Verify we found some symbols
	if len(symbols) == 0 {
		t.Error("No symbols extracted - symbol extraction might not be working")
	}

	// Check for expected symbol types
	var foundInterface, foundClass, foundFunction bool
	for _, symbol := range symbols {
		t.Logf("Found symbol: %s (%s) at line %d", symbol.Name, symbol.Type, symbol.Location.Line)
		
		switch symbol.Type {
		case types.SymbolTypeInterface:
			foundInterface = true
		case types.SymbolTypeClass:
			foundClass = true
		case types.SymbolTypeFunction:
			foundFunction = true
		}
	}

	if !foundInterface {
		t.Error("Expected to find interface symbols")
	}
	if !foundClass {
		t.Error("Expected to find class symbols")
	}
	if !foundFunction {
		t.Error("Expected to find function symbols")
	}

	// Extract imports
	imports, err := manager.ExtractImports(ast)
	if err != nil {
		t.Fatalf("Failed to extract imports: %v", err)
	}

	t.Logf("Extracted %d imports", len(imports))

	// Verify we found imports
	if len(imports) == 0 {
		t.Error("No imports extracted - import extraction might not be working")
	}

	// Check for expected imports
	foundReactImport := false
	for _, imp := range imports {
		t.Logf("Found import: %s from %s", imp.Specifiers, imp.Path)
		if imp.Path == "react" {
			foundReactImport = true
		}
	}

	if !foundReactImport {
		t.Error("Expected to find React import")
	}
}

func TestJavaScriptParsing(t *testing.T) {
	manager := NewManager()

	testContent := `// Sample JavaScript file
const express = require('express');
const { promisify } = require('util');

class Server {
  constructor(port = 3000) {
    this.port = port;
    this.app = express();
  }

  start() {
    this.app.listen(this.port, () => {
      console.log(` + "`" + `Server running on port ${this.port}` + "`" + `);
    });
  }
}

function createServer(port) {
  return new Server(port);
}

module.exports = { Server, createServer };`

	// Create temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.js")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the file
	lang := manager.detectLanguage(testFile)
	if lang == nil || lang.Name != "javascript" {
		t.Fatal("Failed to detect JavaScript language")
	}

	ast, err := manager.ParseFile(testFile, *lang)
	if err != nil {
		t.Fatalf("Failed to parse JavaScript file: %v", err)
	}

	// Basic verification
	if ast.Root == nil {
		t.Fatal("AST root is nil")
	}

	if len(ast.Root.Children) == 0 {
		t.Error("AST root has no children")
	}

	t.Logf("JavaScript AST parsed successfully with %d top-level nodes", len(ast.Root.Children))

	// Extract symbols
	symbols, err := manager.ExtractSymbols(ast)
	if err != nil {
		t.Fatalf("Failed to extract symbols: %v", err)
	}

	t.Logf("Extracted %d symbols from JavaScript", len(symbols))

	// Verify we found some symbols
	if len(symbols) == 0 {
		t.Error("No symbols extracted from JavaScript")
	}
}

func TestParserPerformance(t *testing.T) {
	manager := NewManager()

	// Create a larger test file to measure performance
	largeContent := `// Large TypeScript file for performance testing
import { Component, useState, useEffect } from 'react';

interface Props {
  title: string;
  items: string[];
}

interface State {
  loading: boolean;
  error: string | null;
}
`

	// Repeat the content to make it larger
	for i := 0; i < 10; i++ {
		largeContent += `
class TestClass` + string(rune('A'+i)) + ` {
  private data: any[] = [];
  
  public async process(): Promise<void> {
    for (let i = 0; i < 1000; i++) {
      this.data.push({ id: i, value: 'test' + i });
    }
  }
  
  public getData(): any[] {
    return this.data;
  }
}

function testFunction` + string(rune('A'+i)) + `(param: string): string {
  return 'processed: ' + param;
}
`
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.ts")
	err := os.WriteFile(testFile, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	lang := manager.detectLanguage(testFile)
	if lang == nil {
		t.Fatal("Failed to detect language")
	}

	// Parse and measure time
	ast, err := manager.ParseFile(testFile, *lang)
	if err != nil {
		t.Fatalf("Failed to parse large file: %v", err)
	}

	if ast.Root == nil {
		t.Fatal("AST root is nil")
	}

	symbols, err := manager.ExtractSymbols(ast)
	if err != nil {
		t.Fatalf("Failed to extract symbols: %v", err)
	}

	t.Logf("Performance test: parsed %d bytes, extracted %d symbols", 
		len(largeContent), len(symbols))

	// Should find multiple classes and functions
	if len(symbols) < 20 {
		t.Errorf("Expected at least 20 symbols, got %d", len(symbols))
	}
}