import { SidebarProvider, SidebarTrigger } from "./components/ui/sidebar";
import { AppSidebar } from "./components/app-sidebar";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import Editor from "./components/editor";

const router = createBrowserRouter([
  {
    path: "/",
    element: <div>Hello world!</div>,
  },
  {
    path: "/editor",
    element: <Editor />,
  },
]);

function App() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <div className="p-4 mt-2 min-w-full">
        <SidebarTrigger className="mt-4" />
        <div className="min-h-screen min-w-full bg-white grid mx-auto py-8">
          <RouterProvider router={router} />
        </div>
      </div>
    </SidebarProvider>
  );
}

export default App;
