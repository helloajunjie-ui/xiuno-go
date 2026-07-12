// Minimal type declarations for @toast-ui/editor v3.2.2
// The package ships its own types but they're not resolvable via package.json exports
declare module '@toast-ui/editor' {
  interface EditorOptions {
    el: HTMLElement
    height?: string
    initialEditType?: 'markdown' | 'wysiwyg'
    previewStyle?: 'tab' | 'vertical'
    initialValue?: string
    placeholder?: string
    language?: string
    hideModeSwitch?: boolean
    hooks?: {
      addImageBlobHook?: (blob: Blob | File, callback: (url: string, altText?: string) => void) => void
    }
  }

  class Editor {
    constructor(options: EditorOptions)
    getMarkdown(): string
    setMarkdown(markdown: string): void
    insertText(text: string): void
    destroy(): void
  }

  export default Editor
}
