import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import { Avatar, AvatarImage, AvatarFallback } from "../ui/avatar";
import useOperationsHook from "@/hooks/useOperationHook";
import { Button } from "../ui/button";
import { RefreshCcw } from "lucide-react";
import { useEffect, useState } from "react";
import BlockEditor from "./BlockEditor";
import DocumentLoader from "./DocumentLoader";

export default function Editor() {
  const [
    StepsTracker,
    sendOperations,
    document,
    setDocument,
    editorView,
    setEditorView,
  ] = useOperationsHook();

  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (document.length === 0) {
      return;
    }

    setEditorView(<p></p>);
    setTimeout(() => {
      setEditorView(
        <BlockEditor
          documentContent={document}
          operationExtension={StepsTracker}
        />
      );
      setLoading(false);
    }, 500);
  }, [document]);

  // Renders the editor instance using a React component.
  return (
    <div className="relative">
      <div className="absolute top-0 right-0">
        <Button onClick={sendOperations} disabled={loading}>
          <RefreshCcw size={24} />
          Sync
        </Button>
      </div>
      <Avatar className="ml-12 w-[75px] h-[75px]">
        <AvatarImage src="https://github.com/shadcn.png" />
        <AvatarFallback>CN</AvatarFallback>
      </Avatar>
      {/* Title of the page */}
      <h1 className="text-4xl font-bold mt-4 ml-12 mb-8">BlockNote Editor</h1>
      {/* Editor component */}
      <div>{loading ? <DocumentLoader /> : editorView}</div>
    </div>
  );
}
