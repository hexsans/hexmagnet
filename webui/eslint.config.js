import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import { reactRefresh } from "eslint-plugin-react-refresh";
import stylistic from "@stylistic/eslint-plugin";
import globals from "globals";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "src/lib/graphql/generated"] },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh.plugin,
      "@stylistic": stylistic,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
      "@stylistic/semi": ["error", "always"],
      "@stylistic/quotes": ["error", "double"],
      "@stylistic/comma-dangle": ["error", "always"],
      "@stylistic/indent": ["error", 2, { SwitchCase: 1 }],
      "@stylistic/arrow-parens": ["error", "always"],
      "@stylistic/linebreak-style": ["error", "unix"],
      "@stylistic/member-delimiter-style": ["error", {
        multiline: { delimiter: "semi", requireLast: true },
        singleline: { delimiter: "semi", requireLast: false },
      }],
      "max-len": ["warn", { code: 140, ignoreUrls: true, ignoreStrings: true, ignoreTemplateLiterals: true }],
      "no-tabs": "error",
    },
  },
);
