import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "../graphql/schema/*.graphqls",
  documents: ["src/lib/graphql/**/*.graphql"],
  ignoreNoDocuments: true,
  generates: {
    "src/lib/graphql/generated/": {
      preset: "client",
      config: {
        fragmentMasking: false,
        scalars: {
          Hash20: "string",
          Date: "string",
          DateTime: "string",
          Duration: "string",
          Void: "void",
          Year: "number",
          Uint64: "number",
        },
      },
    },
  },
};

export default config;
