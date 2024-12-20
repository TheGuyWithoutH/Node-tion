import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import { Avatar, AvatarImage, AvatarFallback } from "../ui/avatar";
import useOperationsHook from "@/hooks/useOperationHook";
import useHistoryHooks from "@/hooks/useHistoryHooks";
import { Button } from "../ui/button";
import { Loader2, RefreshCcw } from "lucide-react";
import { useEffect, useState } from "react";
import BlockEditor from "./BlockEditor";
import DocumentLoader from "./DocumentLoader";
import { History } from "lucide-react";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import { useParams } from "react-router-dom";
import { Badge } from "../ui/badge";

export default function Editor() {
  const { docID } = useParams();
  const [
    StepsTracker,
    sendOperations,
    document,
    setDocument,
    editorView,
    setEditorView,
  ] = useOperationsHook(docID || "doc1");
  const [
    historyPoints,
    currentHistoryPoint,
    isViewingHistory,
    setIsViewingHistory,
    currentHistoryPointDocument,
    viewHistoryPoint,
  ] = useHistoryHooks(docID || "doc1");

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

  const historyViewer = useCreateBlockNote({
    initialContent:
      currentHistoryPointDocument.length > 0
        ? currentHistoryPointDocument
        : undefined,
  });

  if (isViewingHistory) {
    return (
      <div className="relative">
        <Button variant={"ghost"} onClick={() => setIsViewingHistory(false)}>
          ← Back
        </Button>
        <div className="flex mt-8">
          <div className="flex-grow">
            <BlockNoteView
              editor={historyViewer}
              theme={"light"}
              editable={false}
            />
          </div>
          <div className="w-1/4 px-4">
            <h2 className="text-2xl font-bold mb-4">History Points</h2>
            <div className="flex flex-col gap-4">
              {historyPoints.map((point, index) => (
                <Badge
                  variant={
                    point === currentHistoryPoint ? "default" : "outline"
                  }
                  className="text-sm cursor-pointer"
                  key={index}
                  onClick={() => viewHistoryPoint(point)}
                >
                  {
                    // turn unix timestamp into human readable date
                    new Date(point).toLocaleDateString() +
                      ", " +
                      new Date(point).toLocaleTimeString()
                  }
                </Badge>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  } else {
    return (
      <div className="relative">
        <Avatar className="ml-12 w-[75px] h-[75px]">
          <AvatarImage src="https://github.com/shadcn.png" />
          <AvatarFallback>CN</AvatarFallback>
        </Avatar>
        <div className="absolute top-0 right-0 flex gap-4 mt-2 mr-2">
          <Button
            onClick={() => setIsViewingHistory(true)}
            variant={"secondary"}
            disabled={true}
          >
            <History size={60} />
            History
          </Button>
          <Button onClick={sendOperations} disabled={loading}>
            {loading ? (
              <Loader2 className="animate-spin" />
            ) : (
              <RefreshCcw size={24} />
            )}
            Sync
          </Button>
        </div>
        {/* Title of the page */}
        <h1 className="text-4xl font-bold mt-4 ml-12 mb-8">
          {docID === "doc1" ? "BlockNote Editor" : docID}
        </h1>
        {/* Editor component */}
        <div>{loading ? <DocumentLoader /> : editorView}</div>
      </div>
    );
  }
}
