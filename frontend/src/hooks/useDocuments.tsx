import { useEffect, useState } from "react";
import { GetDocumentList } from "../../wailsjs/go/impl/node";
import { toast } from "./use-toast";

const useDocumentsHooks = (navigate: (url: string) => void) => {
  const [documentList, setDocumentList] = useState<string[]>([]);

  useEffect(() => {
    try {
      // Fetch the document list from the backend.
      GetDocumentList().then((data) => {
        setDocumentList(data);
      });
    } catch (error) {
      console.error("Error fetching document list", error);
    }
  }, []);

  const createDocument = (name: string) => {
    // Sanitize the name.
    name = name.replace(/[^a-zA-Z0-9]/g, "");

    // Check if the document name is unique.
    if (name && documentList.indexOf(name) === -1) {
      setDocumentList([...documentList, name]);
      navigate(`/editor/${name}`);
    } else {
      toast({
        title: "Document Already Exists",
        description: "You cannot create a document with the same name.",
      });
    }
  };

  return [documentList, createDocument] as const;
};

export default useDocumentsHooks;
