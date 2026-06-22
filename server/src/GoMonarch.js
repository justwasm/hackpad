// Go Monarch tokenizer — derived from vscode go.tmLanguage.json
// Monaco native tokenizer format, no extra dependencies.

const goKeywords = [
  'break', 'case', 'chan', 'const', 'continue', 'default', 'defer',
  'else', 'fallthrough', 'for', 'func', 'go', 'goto', 'if', 'import',
  'interface', 'map', 'package', 'range', 'return', 'select', 'struct',
  'switch', 'type', 'var',
];

const goTypes = [
  'bool', 'byte', 'complex64', 'complex128', 'error', 'float32', 'float64',
  'int', 'int8', 'int16', 'int32', 'int64', 'rune', 'string', 'uint',
  'uint8', 'uint16', 'uint32', 'uint64', 'uintptr', 'any', 'comparable',
];

const goBuiltins = [
  'append', 'cap', 'close', 'copy', 'delete', 'imag', 'len',
  'make', 'new', 'panic', 'print', 'println', 'real', 'recover',
];

const goConstants = ['true', 'false', 'nil', 'iota'];

export const goLanguage = {
  defaultToken: '',
  tokenPostfix: '.go',

  keywords: goKeywords,
  types: goTypes,
  builtins: goBuiltins,
  constants: goConstants,

  operators: [
    '+', '-', '*', '/', '%', '&', '|', '^', '<<', '>>', '&^',
    '+=', '-=', '*=', '/=', '%=', '&=', '|=', '^=', '<<=', '>>=', '&^=',
    '&&', '||', '<-', '++', '--',
    '==', '!=', '<', '<=', '>', '>=',
    '=', ':=', '!', '...',
  ],

  // regex string for @operators reference
  operatorPattern: '[-+*/%&|^<>=!:]+|<<|>>|&&|\\|\\||<-|\\.\\.\\.',

  brackets: [
    { open: '{', close: '}', token: 'delimiter.curly' },
    { open: '[', close: ']', token: 'delimiter.square' },
    { open: '(', close: ')', token: 'delimiter.parenthesis' },
  ],

  // helper: word boundary
  escapes: /\\(?:[abfnrtv\\"']|x[0-9A-Fa-f]{2}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})/,

  tokenizer: {
    root: [
      // identifiers and keywords
      [/[a-zA-Z_]\w*/, {
        cases: {
          '@keywords': 'keyword',
          '@types': 'type',
          '@builtins': 'builtin',
          '@constants': 'constant.language',
          '@default': 'identifier',
        }
      }],

      // package name after "package"
      [/package\s+(\w+)/, ['keyword', { token: 'entity.name.package', next: '@push' }]],

      // function name after "func"
      [/func\s+(\w+)/, ['keyword', 'entity.name.function']],

      // type name after "type"
      [/type\s+(\w+)/, ['keyword', 'entity.name.type']],

      // struct/interface after "type name"
      [/(type\s+\w+\s+)(struct|interface)\b/, ['@brackets', 'keyword']],

      // method receiver like (t *Type)
      [/\b(\w+)\s+\*?(\w+)\)/, ['variable.parameter', 'type']],

      // import paths
      [/"([^"]+)"/, 'string.quoted'],

      // strings
      [/`[^`]*`/, 'string.quoted.backtick'],
      [/"/, { token: 'string.quoted.double', next: '@string' }],

      // rune literals
      [/'[^'\\]'/, 'string.quoted.single'],
      [/'\\u[0-9A-Fa-f]{4}'/, 'string.quoted.single'],

      // comments
      [/\/\/.*$/, 'comment'],
      [/\/\*/, { token: 'comment.block', next: '@comment' }],

      // numbers
      [/\d+[i]/, 'number.float'],
      [/\d+\.\d*([eE][+-]?\d+)?[i]?/, 'number.float'],
      [/0[xX][0-9A-Fa-f_]+/, 'number.hex'],
      [/0[oO][0-7_]+/, 'number.octal'],
      [/0[bB][01_]+/, 'number.binary'],
      [/\d+/, 'number'],

      // operators
      [/[{}()\[\]]/, '@brackets'],
      [/@operatorPattern/, 'operator'],

      // whitespace
      [/\s+/, 'white'],
    ],

    string: [
      [/[^\\"]+/, 'string.quoted.double'],
      [/@escapes/, 'string.escape'],
      [/\\./, 'string.escape.invalid'],
      [/"/, { token: 'string.quoted.double', next: '@pop' }],
    ],

    comment: [
      [/[^/*]+/, 'comment.block'],
      [/\*\//, { token: 'comment.block', next: '@pop' }],
      [/[/*]/, 'comment.block'],
    ],
  },
};

export function registerGoLanguage(monaco) {
  monaco.languages.register({ id: 'go' });
  monaco.languages.setMonarchTokensProvider('go', goLanguage);
}
