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

  operatorPattern: '[-+*/%&|^<>=!:]+|<<|>>|&&|\\|\\||<-|\\.\\.\\.',

  brackets: [
    { open: '{', close: '}', token: 'delimiter.curly' },
    { open: '[', close: ']', token: 'delimiter.square' },
    { open: '(', close: ')', token: 'delimiter.parenthesis' },
  ],

  escapes: /\\(?:[abfnrtv\\"']|x[0-9A-Fa-f]{2}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})/,

  tokenizer: {
    root: [
      [/[a-zA-Z_]\w*/, {
        cases: {
          '@keywords': 'keyword',
          '@types': 'type',
          '@builtins': 'builtin',
          '@constants': 'constant.language',
          '@default': 'identifier',
        }
      }],

      [/package\s+(\w+)/, ['keyword', { token: 'entity.name.package', next: '@push' }]],
      [/func\s+(\w+)/, ['keyword', 'entity.name.function']],
      [/type\s+(\w+)/, ['keyword', 'entity.name.type']],
      [/(type\s+\w+\s+)(struct|interface)\b/, ['@brackets', 'keyword']],
      [/\b(\w+)\s+\*?(\w+)\)/, ['variable.parameter', 'type']],
      [/"([^"]+)"/, 'string.quoted'],
      [/`[^`]*`/, 'string.quoted.backtick'],
      [/"/, { token: 'string.quoted.double', next: '@string' }],
      [/'[^'\\]'/, 'string.quoted.single'],
      [/'\\u[0-9A-Fa-f]{4}'/, 'string.quoted.single'],
      [/\/\/.*$/, 'comment'],
      [/\/\*/, { token: 'comment.block', next: '@comment' }],
      [/\d+[i]/, 'number.float'],
      [/\d+\.\d*([eE][+-]?\d+)?[i]?/, 'number.float'],
      [/0[xX][0-9A-Fa-f_]+/, 'number.hex'],
      [/0[oO][0-7_]+/, 'number.octal'],
      [/0[bB][01_]+/, 'number.binary'],
      [/\d+/, 'number'],
      [/[{}()\[\]]/, '@brackets'],
      [/@operatorPattern/, 'operator'],
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
