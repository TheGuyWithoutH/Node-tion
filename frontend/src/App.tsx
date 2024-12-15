import { SidebarProvider, SidebarTrigger } from "./components/ui/sidebar";
import { AppSidebar } from "./components/app-sidebar";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import Editor from "./components/editor";
import Home from "./components/home";
import Peer from "./components/peer";

const router = createBrowserRouter([
  {
    path: "/",
    element: <Home />,
  },
  {
    path: "/editor",
    element: <Editor />,
  },
  {
    path: "/add-peer",
    element: <Peer />,
  },
]);

function App() {
  return (
    <SidebarProvider className="flex flex-row w-full h-full">
      <AppSidebar />
      <div className="p-4 mt-2 w-auto flex justify-center flex-1">
        <SidebarTrigger className="mt-4" />
        <div className="w-full bg-white grid mx-auto py-8 max-w-[800px]">
          <RouterProvider router={router} />
        </div>
      </div>
    </SidebarProvider>
  );
}

export default App;
