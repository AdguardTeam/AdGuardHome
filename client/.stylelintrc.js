module.exports = {
  rules: {
      "selector-type-no-unknown": true,
      "block-closing-brace-empty-line-before": "never",
      "block-no-empty": true,
      "block-opening-brace-newline-after": "always",
      "block-opening-brace-space-before": "always",
      "color-hex-case": "lower",
      "color-named": "never",
      "color-no-invalid-hex": true,
      "length-zero-no-unit": true,
      "declaration-block-trailing-semicolon": "always",
      "custom-property-empty-line-before": ["always", {
          "except": [
              "after-custom-property",
              "first-nested"
          ]
      }],
      "declaration-block-no-duplicate-properties": true,
      "declaration-colon-space-after": "always",
      "declaration-empty-line-before": ["always", {
          "except": [
              "after-declaration",
              "first-nested",
              "after-comment"
          ]
      }],
      "font-weight-notation": "numeric",
      "indentation": [4, {
          "except": ["value"]
      }],
      "max-empty-lines": 2,
      "no-missing-end-of-source-newline": true,
      "number-leading-zero": "always",
      "property-no-unknown": true,
      "rule-empty-line-before": ["always-multi-line", {
          "except": ["first-nested"],
          "ignore": ["after-comment"]
      }],
      "string-quotes": "double",
      "value-list-comma-space-after": "always",
      "unit-case": "lower"
  }
}
