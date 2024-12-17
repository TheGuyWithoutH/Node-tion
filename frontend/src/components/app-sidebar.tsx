import {
  ChevronDown,
  ChevronsUpDown,
  ChevronUp,
  Edit,
  Home,
  Plus,
  User2Icon,
} from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { Avatar, AvatarFallback } from "./ui/avatar";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@radix-ui/react-collapsible";
import { useState } from "react";

export function AppSidebar({
  documentList,
  onNewDocument,
}: {
  documentList: string[];
  onNewDocument: () => void;
}) {
  // Menu items.
  const items = [
    {
      title: "Home",
      url: "/",
      icon: Home,
    },
    {
      title: "Editor",
      url: "/editor",
      icon: Edit,
      children: [
        {
          title: "New Document",
          icon: Plus,
          action: onNewDocument,
        },
        {
          title: "Test Document",
          url: "/editor/doc1",
        },
        ...documentList.map((doc) => ({
          title: doc.slice(0, 10),
          url: `/editor/${doc}`,
        })),
      ],
    },
    {
      title: "Add Peer",
      url: "/add-peer",
      icon: User2Icon,
    },
  ];

  return (
    <Sidebar>
      <SidebarHeader className="pt-6">
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton className="flex items-center h-100">
                  <Avatar>
                    {/* <AvatarImage src="https://github.com/shadcn.png" /> */}
                    <AvatarFallback>BL</AvatarFallback>
                  </Avatar>

                  <div className="flex flex-col">
                    Select Workspace
                    <p className="text-xs text-gray-500">Acme Inc</p>
                  </div>
                  <ChevronsUpDown className="ml-auto" />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent className="w-[--radix-popper-anchor-width]">
                <DropdownMenuItem>
                  <span>Project 1</span>
                </DropdownMenuItem>
                <DropdownMenuItem>
                  <span>Project 2</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Application</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => {
                if (item.children) {
                  return <CollapsibleMenu key={item.title} item={item} />;
                } else {
                  return (
                    <SidebarMenuItem key={item.title}>
                      <SidebarMenuButton asChild>
                        <a href={item.url}>
                          <item.icon />
                          <span>{item.title}</span>
                        </a>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                }
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}

const CollapsibleMenu = ({
  item,
}: {
  item: {
    title: string;
    url: string;
    icon?: any;
    children: {
      title: string;
      url?: string;
      icon?: any;
      action?: () => void;
    }[];
  };
}) => {
  const [open, setOpen] = useState(false);

  return (
    <Collapsible
      className="group/collapsible"
      open={open}
      onOpenChange={setOpen}
    >
      <SidebarMenuItem>
        <CollapsibleTrigger asChild>
          <SidebarMenuButton asChild>
            <div>
              {item.icon ? <item.icon /> : null}
              <span>{item.title}</span>
              {open ? (
                <ChevronUp className="ml-auto mr-2" />
              ) : (
                <ChevronDown className="ml-auto mr-2" />
              )}
            </div>
          </SidebarMenuButton>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <SidebarMenuSub>
            {item.children.map((subItem) => (
              <SidebarMenuSubItem
                key={subItem.title}
                className="my-0.5 hover:bg-gray-100 p-1"
              >
                {subItem.action ? (
                  <button
                    onClick={subItem.action}
                    className="flex items-center w-full"
                  >
                    <span>{subItem.title}</span>
                    {subItem.icon ? (
                      <subItem.icon size={20} className={"ml-auto"} />
                    ) : null}
                  </button>
                ) : (
                  <a href={subItem.url} className="flex items-center">
                    <span>{subItem.title}</span>
                    {subItem.icon ? (
                      <subItem.icon size={20} className={"ml-auto"} />
                    ) : null}
                  </a>
                )}
              </SidebarMenuSubItem>
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  );
};
