import {
  createBrowserRouter,
  RouterProvider,
  Outlet,
  useNavigate,
} from "react-router-dom";
import { SidebarProvider, SidebarTrigger } from "./components/ui/sidebar";
import { AppSidebar } from "./components/app-sidebar";
import { Toaster } from "./components/ui/toaster";
import { CreateDocumentSheet } from "./components/document/CreateDocumentSheet";
import Editor from "./components/editor";
import Home from "./components/home";
import Peer from "./components/peer";
import useDocumentsHooks from "./hooks/useDocuments";
import { useState } from "react";

// Shared layout that includes the sidebar, sheet, and outlet for routes
function Layout() {
  const [isSheetOpen, setSheetOpen] = useState(false);
  const navigate = useNavigate();
  const [documentList, createDocument] = useDocumentsHooks(navigate);

  return (
    <SidebarProvider className="flex flex-row w-full h-full">
      <Toaster />
      <AppSidebar
        documentList={documentList}
        onNewDocument={() => setSheetOpen(true)}
      />
      <div className="p-4 mt-2 w-auto flex justify-center flex-1">
        <SidebarTrigger className="mt-4" />
        <div className="w-full bg-white grid mx-auto py-8 max-w-[800px]">
          {/* Outlet for child routes */}
          <Outlet />
        </div>
      </div>

      {/* Controlled Create Document Sheet */}
      <CreateDocumentSheet
        isOpen={isSheetOpen}
        onOpenChange={setSheetOpen}
        onDocumentCreate={createDocument}
      />
    </SidebarProvider>
  );
}

// Define routes with the shared layout
const router = createBrowserRouter([
  {
    path: "/",
    element: <Layout />, // Use the layout here
    children: [
      { index: true, element: <Home /> }, // Default child route
      { path: "editor/:docID", element: <Editor /> },
      { path: "add-peer", element: <Peer /> },
    ],
  },
]);

function App() {
  return <RouterProvider router={router} />;
}

export default App;
