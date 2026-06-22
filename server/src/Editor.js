import * as monaco from 'monaco-editor';
import { listenColorScheme } from './ColorScheme';
import './Editor.css';

const MONACO_CDN = 'https://cdn.jsdelivr.net/npm/monaco-editor@0.55.1/min/vs';

window.MonacoEnvironment = {
  getWorker(_, label) {
    if (label === 'json') return new Worker(`${MONACO_CDN}/language/json/json.worker.js`);
    if (label === 'css' || label === 'scss' || label === 'less') return new Worker(`${MONACO_CDN}/language/css/css.worker.js`);
    if (label === 'html' || label === 'handlebars' || label === 'razor') return new Worker(`${MONACO_CDN}/language/html/html.worker.js`);
    if (label === 'typescript' || label === 'javascript') return new Worker(`${MONACO_CDN}/language/typescript/ts.worker.js`);
    return new Worker(`${MONACO_CDN}/editor/editor.worker.js`);
  }
};

const THEME_LIGHT = 'vs';
const THEME_DARK = 'vs-dark';

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

  return {
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
  };
}
