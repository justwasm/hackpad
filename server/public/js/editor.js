import { listenColorScheme } from './colorScheme.js';
import { registerGoLanguage } from './goMonarch.js';

const THEME_LIGHT = 'vs';
const THEME_DARK = 'vs-dark';

export function setupMonaco(monaco) {
  registerGoLanguage(monaco);

  const WORKER_BASE = 'vs/assets';

  window.MonacoEnvironment = {
    getWorker(_, label) {
      if (label === 'json') return new Worker(`${WORKER_BASE}/json.worker.js`);
      if (label === 'css' || label === 'scss' || label === 'less') return new Worker(`${WORKER_BASE}/css.worker.js`);
      if (label === 'html' || label === 'handlebars' || label === 'razor') return new Worker(`${WORKER_BASE}/html.worker.js`);
      if (label === 'typescript' || label === 'javascript') return new Worker(`${WORKER_BASE}/ts.worker.js`);
      return new Worker(`${WORKER_BASE}/editor.worker.js`);
    }
  };
}

export function newEditor(elem, onEdit, monaco) {
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
    setLanguage(lang) {
      monaco.editor.setModelLanguage(editor.getModel(), lang);
    },
  };
}
