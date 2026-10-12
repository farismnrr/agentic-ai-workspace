import js from '@eslint/js';
import { defineConfig } from 'eslint/config';
import globals from 'globals';
import svelte from 'eslint-plugin-svelte';
import ts from 'typescript-eslint';

export default defineConfig(
  {
    ignores: ['node_modules/**'],
  },
  js.configs.recommended,
  ts.configs.strictTypeChecked,
  ts.configs.stylisticTypeChecked,
  svelte.configs.recommended,
  {
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.svelte'],
      },
    },
    rules: {
      complexity: ['error', 10],
      'max-lines': [
        'error',
        { max: 200, skipBlankLines: false, skipComments: false },
      ],
      'max-lines-per-function': [
        'error',
        { max: 60, skipBlankLines: false, skipComments: false },
      ],
      '@typescript-eslint/consistent-type-imports': 'error',
      '@typescript-eslint/no-explicit-any': 'error',
    },
  },
  {
    files: ['src/**/*.{ts,svelte}'],
    languageOptions: {
      globals: globals.browser,
    },
  },
  {
    files: ['**/*.svelte'],
    languageOptions: {
      parserOptions: {
        parser: ts.parser,
      },
    },
  },
  {
    files: ['vite.config.ts', 'eslint.config.ts'],
    languageOptions: {
      globals: globals.nodeBuiltin,
    },
  },
);
