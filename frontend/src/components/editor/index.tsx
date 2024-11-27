import "@blocknote/core/fonts/inter.css";
import { BlockNoteView } from "@blocknote/mantine";
import "@blocknote/mantine/style.css";
import { useCreateBlockNote } from "@blocknote/react";
import { Avatar, AvatarImage, AvatarFallback } from "../ui/avatar";

export default function Editor() {
  // Creates a new editor instance.
  const editor = useCreateBlockNote({
    initialContent: [
      {
        type: "paragraph",
        content: "Welcome to this demo!",
      },
      {
        type: "paragraph",
        content: "<- Notice the new button in the side menu",
      },
      {
        type: "paragraph",
        content: "Click it to remove the hovered block",
      },
      {
        type: "paragraph",
      },
    ],
  });

  // Renders the editor instance using a React component.
  return (
    <div className="relative">
      <Avatar className="ml-12 w-[75px] h-[75px]">
        <AvatarImage src="https://github.com/shadcn.png" />
        <AvatarFallback>CN</AvatarFallback>
      </Avatar>
      {/* Title of the page */}
      <h1 className="text-4xl font-bold mt-4 ml-12 mb-8">BlockNote Editor</h1>
      {/* Editor component */}
      <BlockNoteView editor={editor} theme={"light"} />
    </div>
  );
}
