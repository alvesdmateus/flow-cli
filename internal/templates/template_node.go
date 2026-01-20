package templates

var nodeTemplate = Template{
	Name:        "node",
	Language:    "JavaScript/TypeScript",
	Description: "Node.js project with TypeScript support",
	Features: []string{
		"TypeScript configuration",
		"ESLint + Prettier for code quality",
		"Jest for testing",
		"npm scripts for common tasks",
		".vibe configuration",
	},
	Files: []TemplateFile{
		{
			Path: "package.json",
			Content: `{
  "name": "{{.Name}}",
  "version": "0.1.0",
  "description": "{{.Description}}",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "scripts": {
    "build": "tsc",
    "dev": "ts-node src/index.ts",
    "start": "node dist/index.js",
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage",
    "lint": "eslint src/ --ext .ts",
    "lint:fix": "eslint src/ --ext .ts --fix",
    "format": "prettier --write \"src/**/*.ts\"",
    "format:check": "prettier --check \"src/**/*.ts\"",
    "clean": "rm -rf dist/ coverage/"
  },
  "keywords": [],
  "author": "{{.Author}}",
  "license": "MIT",
  "devDependencies": {
    "@types/jest": "^29.5.0",
    "@types/node": "^20.0.0",
    "@typescript-eslint/eslint-plugin": "^6.0.0",
    "@typescript-eslint/parser": "^6.0.0",
    "eslint": "^8.0.0",
    "eslint-config-prettier": "^9.0.0",
    "jest": "^29.0.0",
    "prettier": "^3.0.0",
    "ts-jest": "^29.0.0",
    "ts-node": "^10.9.0",
    "typescript": "^5.0.0"
  }
}
`,
		},
		{
			Path: "tsconfig.json",
			Content: `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs",
    "lib": ["ES2022"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "resolveJsonModule": true,
    "moduleResolution": "node"
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "coverage"]
}
`,
		},
		{
			Path: ".eslintrc.json",
			Content: `{
  "root": true,
  "parser": "@typescript-eslint/parser",
  "plugins": ["@typescript-eslint"],
  "extends": [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "prettier"
  ],
  "env": {
    "node": true,
    "es2022": true,
    "jest": true
  },
  "rules": {
    "@typescript-eslint/explicit-function-return-type": "warn",
    "@typescript-eslint/no-unused-vars": ["error", { "argsIgnorePattern": "^_" }]
  }
}
`,
		},
		{
			Path: ".prettierrc",
			Content: `{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2
}
`,
		},
		{
			Path: "jest.config.js",
			Content: `/** @type {import('jest').Config} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/src'],
  testMatch: ['**/*.test.ts'],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
    '!src/**/*.test.ts',
  ],
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'html'],
};
`,
		},
		{
			Path: "src/index.ts",
			Content: `/**
 * {{.Name}}
 * {{.Description}}
 */

export function greet(name: string): string {
  return ` + "`Hello, ${name}!`" + `;
}

function main(): void {
  console.log(greet('{{.Name}}'));
}

// Run if executed directly
if (require.main === module) {
  main();
}
`,
		},
		{
			Path: "src/index.test.ts",
			Content: `import { greet } from './index';

describe('greet', () => {
  it('should return greeting with name', () => {
    expect(greet('World')).toBe('Hello, World!');
  });

  it('should work with different names', () => {
    expect(greet('{{.Name}}')).toBe('Hello, {{.Name}}!');
  });
});
`,
		},
		{
			Path: ".gitignore",
			Content: `# Dependencies
node_modules/

# Build output
dist/

# Coverage
coverage/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Environment
.env
.env.local
.env.*.local

# Logs
logs/
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Runtime data
pids/
*.pid
*.seed
*.pid.lock

# Optional npm cache directory
.npm

# Optional eslint cache
.eslintcache
`,
		},
		{
			Path: "README.md",
			Content: `# {{.Name}}

{{.Description}}

## Getting Started

### Prerequisites

- Node.js 18 or later
- npm 9 or later

### Installation

` + "```bash" + `
npm install
` + "```" + `

### Development

` + "```bash" + `
# Run in development mode
npm run dev

# Build for production
npm run build

# Run production build
npm start
` + "```" + `

### Testing

` + "```bash" + `
# Run tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
` + "```" + `

### Code Quality

` + "```bash" + `
# Lint code
npm run lint

# Fix lint issues
npm run lint:fix

# Format code
npm run format
` + "```" + `

## Project Structure

` + "```" + `
{{.Name}}/
├── src/
│   ├── index.ts        # Main entry point
│   └── index.test.ts   # Tests
├── dist/               # Compiled output
├── package.json
├── tsconfig.json
├── .vibe/
│   └── config.yaml
└── .vibeignore
` + "```" + `

## License

MIT
`,
		},
		{
			Path: ".vibe/config.yaml",
			Content: `# Vibe CLI Project Configuration
# This file contains project-specific settings for vibe-cli

# Project metadata
project:
  name: "{{.Name}}"
  description: "{{.Description}}"
  language: typescript

# Security settings
security:
  trusted_paths: []
  denied_paths:
    - node_modules/
    - dist/
    - coverage/

# Commands specific to this project
commands:
  allowed:
    - npm install
    - npm run
    - npm test
    - npm start
    - npx
    - node
`,
		},
		{
			Path: ".vibeignore",
			Content: `# Files and directories to exclude from vibe indexing

# Dependencies
node_modules/

# Build output
dist/

# Coverage
coverage/

# Lock files (large, auto-generated)
package-lock.json
yarn.lock
pnpm-lock.yaml

# IDE and editor files
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Logs
*.log

# Cache
.eslintcache
.npm/
`,
		},
	},
}
