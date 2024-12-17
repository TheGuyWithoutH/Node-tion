import { PartialBlock } from "@blocknote/core";
import { useEffect, useState } from "react";

const useHistoryHooks = (documentId: string) => {
  const [historyPoints, setHistoryPoints] = useState<number[]>([
    // Unix timestamps
    1734406119000, 1734306119000, 1734206119000, 1633702800000,
  ]);
  const [currentHistoryPoint, setCurrentHistoryPoint] =
    useState<number>(1734406119000);
  const [isViewingHistory, setIsViewingHistory] = useState<boolean>(false);
  const [currentHistoryPointDocument, setCurrentHistoryPointDocument] =
    useState<PartialBlock[]>([
      {
        type: "paragraph",
        content: "Welcome to this demo!",
      },
    ]);

  useEffect(() => {
    if (isViewingHistory) {
      // Fetch the history points from the server.
      //   fetch(`/api/history/${documentId}`)
      //     .then((res) => res.json())
      //     .then((data) => {
      //       setHistoryPoints(data);
      //     });
    }
  }, [isViewingHistory]);

  const viewHistoryPoint = (historyPoint: number) => {
    console.log("Viewing history point", historyPoint);
    setCurrentHistoryPoint(historyPoint);
    // fetch(`/api/history/${documentId}/${historyPointId}`)
    //   .then((res) => res.json())
    //   .then((data) => {
    //     setCurrentHistoryPointDocument(data);
    //   });
  };

  useEffect(() => {
    setCurrentHistoryPointDocument([
      {
        type: "paragraph",
        // @ts-ignore
        content: "Welcome to this demo!",
      },
      {
        type: "paragraph",
        // @ts-ignore
        content: "<- Notice the new button in the side menu",
      },
      {
        type: "paragraph",
        // @ts-ignore
        content: "Click it to remove the hovered block",
      },
    ]);
  }, []);

  return [
    historyPoints,
    currentHistoryPoint,
    isViewingHistory,
    setIsViewingHistory,
    currentHistoryPointDocument,
    viewHistoryPoint,
  ] as const;
};

export default useHistoryHooks;
