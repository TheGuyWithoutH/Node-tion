import { PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import { Extension } from "@tiptap/core";

const BlockEditor = ({
  documentContent,
  operationExtension,
}: {
  documentContent: PartialBlock[];
  operationExtension: Extension<any, any>;
}) => {
  // Creates a new editor instance.
  const editor = useCreateBlockNote({
    initialContent: documentContent,
    trailingBlock: false,
    _tiptapOptions: {
      extensions: [operationExtension],
    },
  });

  // Renders the editor instance using a React component.
  return (
    <BlockNoteView
      editor={editor}
      theme={"light"}
      onChange={() => {
        console.log(editor.document);
      }}
    />
  );
};

export default BlockEditor;
