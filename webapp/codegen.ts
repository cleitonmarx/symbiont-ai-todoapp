import { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
  schema: '../api/graphql/schema.graphql',
  documents: './src/graphql/*.graphql',
  generates: {
    './src/types/graphql.ts': {
      plugins: [
        'typescript',           // Generates base types (enums, inputs, etc.)
        'typescript-operations',// Generates types for your specific queries
        'typed-document-node'   // Bundles the AST with types for auto-inference
      ],
      config: {
        enumsAsTypes: true,         // (already present) Enums as union types
        useTypeImports: true,       // (already present) Use `import type`
        maybeValue: 'T | undefined',// Make nullable fields `T | undefined` (optional, but common in TS projects)
        avoidOptionals: true,       // Makes all fields required unless nullable in schema
        preResolveTypes: true,      // Flattens fragments/types for easier usage
      }
    }
  }
};

export default config;