package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAnalyzer(t *testing.T) {
	a := NewAnalyzer()

	if a == nil {
		t.Fatal("NewAnalyzer returned nil")
	}

	// Check that parsers are registered
	extensions := []string{".go", ".py", ".js", ".ts", ".rs"}
	for _, ext := range extensions {
		if a.GetParser(ext) == nil {
			t.Errorf("No parser registered for %s", ext)
		}
	}
}

func TestGoParser_Parse(t *testing.T) {
	content := []byte(`package main

import (
	"fmt"
	"os"
)

// Person represents a person
type Person struct {
	Name string
	Age  int
}

// Greeter defines greeting behavior
type Greeter interface {
	Greet() string
}

const MaxAge = 120

var defaultName = "John"

// NewPerson creates a new person
func NewPerson(name string, age int) *Person {
	return &Person{Name: name, Age: age}
}

// Greet returns a greeting
func (p *Person) Greet() string {
	return fmt.Sprintf("Hello, I'm %s", p.Name)
}

func main() {
	p := NewPerson("Alice", 30)
	fmt.Println(p.Greet())
}
`)

	parser := &GoParser{}
	outline, err := parser.Parse("test.go", content)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if outline.Package != "main" {
		t.Errorf("Expected package 'main', got '%s'", outline.Package)
	}

	if len(outline.Imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(outline.Imports))
	}

	// Check symbols
	symbolNames := make(map[string]Symbol)
	for _, sym := range outline.Symbols {
		symbolNames[sym.Name] = sym
	}

	// Check Person struct
	if sym, ok := symbolNames["Person"]; !ok {
		t.Error("Person struct not found")
	} else {
		if sym.Kind != SymbolStruct {
			t.Errorf("Person should be struct, got %s", sym.Kind)
		}
		if !sym.Exported {
			t.Error("Person should be exported")
		}
	}

	// Check Greeter interface
	if sym, ok := symbolNames["Greeter"]; !ok {
		t.Error("Greeter interface not found")
	} else {
		if sym.Kind != SymbolInterface {
			t.Errorf("Greeter should be interface, got %s", sym.Kind)
		}
	}

	// Check NewPerson function
	if sym, ok := symbolNames["NewPerson"]; !ok {
		t.Error("NewPerson function not found")
	} else {
		if sym.Kind != SymbolFunction {
			t.Errorf("NewPerson should be function, got %s", sym.Kind)
		}
		if !strings.Contains(sym.Signature, "NewPerson") {
			t.Errorf("Signature should contain 'NewPerson': %s", sym.Signature)
		}
	}

	// Check Greet method
	if sym, ok := symbolNames["Greet"]; !ok {
		t.Error("Greet method not found")
	} else {
		if sym.Kind != SymbolMethod {
			t.Errorf("Greet should be method, got %s", sym.Kind)
		}
		if sym.Parent != "*Person" {
			t.Errorf("Greet parent should be '*Person', got '%s'", sym.Parent)
		}
	}

	// Check const
	if sym, ok := symbolNames["MaxAge"]; !ok {
		t.Error("MaxAge const not found")
	} else {
		if sym.Kind != SymbolConst {
			t.Errorf("MaxAge should be const, got %s", sym.Kind)
		}
	}

	// Check var
	if sym, ok := symbolNames["defaultName"]; !ok {
		t.Error("defaultName var not found")
	} else {
		if sym.Kind != SymbolVar {
			t.Errorf("defaultName should be var, got %s", sym.Kind)
		}
		if sym.Exported {
			t.Error("defaultName should not be exported")
		}
	}
}

func TestPythonParser_Parse(t *testing.T) {
	content := []byte(`import os
from typing import List

class Person:
    def __init__(self, name: str, age: int):
        self.name = name
        self.age = age

    def greet(self) -> str:
        return f"Hello, I'm {self.name}"

def create_person(name: str, age: int) -> Person:
    return Person(name, age)

def main():
    p = create_person("Alice", 30)
    print(p.greet())
`)

	parser := &PythonParser{}
	outline, err := parser.Parse("test.py", content)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if outline.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", outline.Language)
	}

	symbolNames := make(map[string]Symbol)
	for _, sym := range outline.Symbols {
		symbolNames[sym.Name] = sym
	}

	// Check Person class
	if sym, ok := symbolNames["Person"]; !ok {
		t.Error("Person class not found")
	} else {
		if sym.Kind != SymbolClass {
			t.Errorf("Person should be class, got %s", sym.Kind)
		}
	}

	// Check __init__ method
	if sym, ok := symbolNames["__init__"]; !ok {
		t.Error("__init__ method not found")
	} else {
		if sym.Kind != SymbolMethod {
			t.Errorf("__init__ should be method, got %s", sym.Kind)
		}
		if sym.Parent != "Person" {
			t.Errorf("__init__ parent should be 'Person', got '%s'", sym.Parent)
		}
	}

	// Check create_person function
	if sym, ok := symbolNames["create_person"]; !ok {
		t.Error("create_person function not found")
	} else {
		if sym.Kind != SymbolFunction {
			t.Errorf("create_person should be function, got %s", sym.Kind)
		}
	}
}

func TestJavaScriptParser_Parse(t *testing.T) {
	content := []byte(`import React from 'react';
import { useState } from 'react';

export class Person {
    constructor(name, age) {
        this.name = name;
        this.age = age;
    }

    greet() {
        return "Hello, I'm " + this.name;
    }
}

export function createPerson(name, age) {
    return new Person(name, age);
}

const sayHello = () => {
    console.log("Hello!");
};

export const add = (a, b) => a + b;
`)

	parser := &JavaScriptParser{}
	outline, err := parser.Parse("test.js", content)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	symbolNames := make(map[string]Symbol)
	for _, sym := range outline.Symbols {
		symbolNames[sym.Name] = sym
	}

	// Check Person class
	if sym, ok := symbolNames["Person"]; !ok {
		t.Error("Person class not found")
	} else {
		if sym.Kind != SymbolClass {
			t.Errorf("Person should be class, got %s", sym.Kind)
		}
		if !sym.Exported {
			t.Error("Person should be exported")
		}
	}

	// Check createPerson function
	if sym, ok := symbolNames["createPerson"]; !ok {
		t.Error("createPerson function not found")
	} else {
		if sym.Kind != SymbolFunction {
			t.Errorf("createPerson should be function, got %s", sym.Kind)
		}
	}

	// Check imports
	if len(outline.Imports) < 1 {
		t.Error("Expected at least 1 import")
	}
}

func TestTypeScriptParser_Parse(t *testing.T) {
	content := []byte(`import { Component } from 'react';

export interface Person {
    name: string;
    age: number;
}

export type PersonList = Person[];

export class PersonService {
    getPerson(id: string): Person {
        return { name: "Test", age: 30 };
    }
}

export function createPerson(name: string, age: number): Person {
    return { name, age };
}
`)

	parser := &TypeScriptParser{}
	outline, err := parser.Parse("test.ts", content)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if outline.Language != "typescript" {
		t.Errorf("Expected language 'typescript', got '%s'", outline.Language)
	}

	symbolNames := make(map[string]Symbol)
	for _, sym := range outline.Symbols {
		symbolNames[sym.Name] = sym
	}

	// Check Person interface
	if sym, ok := symbolNames["Person"]; !ok {
		t.Error("Person interface not found")
	} else {
		if sym.Kind != SymbolInterface {
			t.Errorf("Person should be interface, got %s", sym.Kind)
		}
	}

	// Check PersonList type
	if sym, ok := symbolNames["PersonList"]; !ok {
		t.Error("PersonList type not found")
	} else {
		if sym.Kind != SymbolType {
			t.Errorf("PersonList should be type, got %s", sym.Kind)
		}
	}
}

func TestRustParser_Parse(t *testing.T) {
	content := []byte(`use std::fmt;
use crate::utils::helper;

pub struct Person {
    pub name: String,
    age: u32,
}

pub enum Status {
    Active,
    Inactive,
}

pub trait Greeter {
    fn greet(&self) -> String;
}

impl Person {
    pub fn new(name: String, age: u32) -> Self {
        Person { name, age }
    }

    fn private_method(&self) {
        println!("private");
    }
}

impl Greeter for Person {
    fn greet(&self) -> String {
        format!("Hello, I'm {}", self.name)
    }
}

pub fn create_person(name: &str) -> Person {
    Person::new(name.to_string(), 0)
}
`)

	parser := &RustParser{}
	outline, err := parser.Parse("test.rs", content)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if outline.Language != "rust" {
		t.Errorf("Expected language 'rust', got '%s'", outline.Language)
	}

	symbolNames := make(map[string]Symbol)
	for _, sym := range outline.Symbols {
		symbolNames[sym.Name] = sym
	}

	// Check Person struct
	if sym, ok := symbolNames["Person"]; !ok {
		t.Error("Person struct not found")
	} else {
		if sym.Kind != SymbolStruct {
			t.Errorf("Person should be struct, got %s", sym.Kind)
		}
		if !sym.Exported {
			t.Error("Person should be exported (pub)")
		}
	}

	// Check Status enum
	if sym, ok := symbolNames["Status"]; !ok {
		t.Error("Status enum not found")
	} else {
		if sym.Kind != SymbolType {
			t.Errorf("Status should be type, got %s", sym.Kind)
		}
	}

	// Check Greeter trait
	if sym, ok := symbolNames["Greeter"]; !ok {
		t.Error("Greeter trait not found")
	} else {
		if sym.Kind != SymbolInterface {
			t.Errorf("Greeter should be interface, got %s", sym.Kind)
		}
	}

	// Check create_person function
	if sym, ok := symbolNames["create_person"]; !ok {
		t.Error("create_person function not found")
	} else {
		if sym.Kind != SymbolFunction {
			t.Errorf("create_person should be function, got %s", sym.Kind)
		}
	}

	// Check imports
	if len(outline.Imports) < 2 {
		t.Errorf("Expected at least 2 imports, got %d", len(outline.Imports))
	}
}

func TestAnalyzer_ParseFile(t *testing.T) {
	// Create a temporary Go file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")

	content := []byte(`package test

func Hello() string {
	return "hello"
}
`)

	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	a := NewAnalyzer()
	outline, err := a.ParseFile(tmpFile)

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if outline.Package != "test" {
		t.Errorf("Expected package 'test', got '%s'", outline.Package)
	}

	if len(outline.Symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(outline.Symbols))
	}
}

func TestAnalyzer_ParseFile_UnsupportedExtension(t *testing.T) {
	a := NewAnalyzer()
	_, err := a.ParseFile("test.xyz")

	if err == nil {
		t.Error("Expected error for unsupported extension")
	}
}

func TestFindSymbol(t *testing.T) {
	outlines := []*FileOutline{
		{
			File: "a.go",
			Symbols: []Symbol{
				{Name: "Foo", Kind: SymbolFunction},
				{Name: "Bar", Kind: SymbolStruct},
			},
		},
		{
			File: "b.go",
			Symbols: []Symbol{
				{Name: "Foo", Kind: SymbolMethod},
				{Name: "Baz", Kind: SymbolVar},
			},
		},
	}

	// Find Foo (should find 2)
	results := FindSymbol(outlines, "Foo")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'Foo', got %d", len(results))
	}

	// Find Bar (should find 1)
	results = FindSymbol(outlines, "Bar")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Bar', got %d", len(results))
	}

	// Find NonExistent (should find 0)
	results = FindSymbol(outlines, "NonExistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'NonExistent', got %d", len(results))
	}
}

func TestSymbolKind_Constants(t *testing.T) {
	// Verify symbol kinds are distinct strings
	kinds := []SymbolKind{
		SymbolFunction, SymbolMethod, SymbolType, SymbolStruct,
		SymbolInterface, SymbolConst, SymbolVar, SymbolClass, SymbolImport,
	}

	seen := make(map[SymbolKind]bool)
	for _, kind := range kinds {
		if seen[kind] {
			t.Errorf("Duplicate symbol kind: %s", kind)
		}
		seen[kind] = true
	}
}
