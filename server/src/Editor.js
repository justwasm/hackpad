import * as monaco from 'monaco-editor';
import { listenColorScheme } from './ColorScheme';
import { registerGoLanguage } from './GoMonarch';
import './Editor.css';

registerGoLanguage(monaco);

const WORKER_BASE = 'https://cdn.jsdelivr.net/npm/monaco-editor@0.55.1/esm/vs';

window.MonacoEnvironment = {
  getWorker(_, label) {
    const opts = { type: 'module' };
    if (label === 'json') return new Worker(`${WORKER_BASE}/language/json/json.worker.js`, opts);
    if (label === 'css' || label === 'scss' || label === 'less') return new Worker(`${WORKER_BASE}/language/css/css.worker.js`, opts);
    if (label === 'html' || label === 'handlebars' || label === 'razor') return new Worker(`${WORKER_BASE}/language/html/html.worker.js`, opts);
    if (label === 'typescript' || label === 'javascript') return new Worker(`${WORKER_BASE}/language/typescript/ts.worker.js`, opts);
    return new Worker(`${WORKER_BASE}/editor/editor.worker.js`, opts);
  }
};

const THEME_LIGHT = 'vs';
const THEME_DARK = 'vs-dark';

// Holds the editor API object; Go adds getCompletions to it after creation.
let editorApi = null;

export function newEditor(elem, onEdit) {
  const editor = monaco.editor.create(elem, {
    language: 'plaintext',
    theme: THEME_DARK,
    automaticLayout: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    fontSize: 14,
    tabSize: 4,
    insertSpaces: false,
    wordWrap: 'off',
    lineNumbersMinChars: 3,
    glyphMargin: false,
    folding: false,
  });

  let ignoreChange = false;
  editor.onDidChangeModelContent(() => {
    if (!ignoreChange) {
      onEdit();
    }
  });

  listenColorScheme({
    light: () => editor.updateOptions({ theme: THEME_LIGHT }),
    dark: () => editor.updateOptions({ theme: THEME_DARK }),
  });

  // Register completion provider for Go
  const completionDisposable = monaco.languages.registerCompletionItemProvider('go', {
    provideCompletionItems: (model, position) => {
      if (typeof editorApi?.getCompletions !== 'function') {
        return { suggestions: [] };
      }
      const content = model.getValue();
      const word = model.getWordAtPosition(position);
      const items = editorApi.getCompletions(content, position.lineNumber, position.column);
      if (!items || !items.length) {
        return { suggestions: [] };
      }
      return {
        suggestions: items.map(item => ({
          label: item.label,
          kind: item.kind,
          detail: item.detail,
          insertText: item.insertText,
          range: word ? {
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: word.startColumn,
            endColumn: word.endColumn,
          } : {
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: position.column,
            endColumn: position.column,
          },
        })),
      };
    },
    triggerCharacters: ['.'],
  });

  editorApi = {
    getContents() {
      return editor.getValue();
    },
    setContents(contents) {
      ignoreChange = true;
      editor.setValue(contents);
      ignoreChange = false;
    },
    getCursorIndex() {
      const pos = editor.getPosition();
      return pos ? pos.column : 0;
    },
    setCursorIndex(index) {
      editor.setPosition({ lineNumber: 1, column: index });
    },
    setLanguage(lang) {
      monaco.editor.setModelLanguage(editor.getModel(), lang);
    },
    dispose() {
      completionDisposable.dispose();
      editor.dispose();
    },
  };
  return editorApi;
}
