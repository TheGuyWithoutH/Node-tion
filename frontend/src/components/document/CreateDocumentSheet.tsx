import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { useState } from "react";

export function CreateDocumentSheet({
  isOpen,
  onOpenChange,
  onDocumentCreate,
}: {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onDocumentCreate: (name: string) => void;
}) {
  const [docName, setDocName] = useState("");

  return (
    <Sheet key={"left"} open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent side={"left"}>
        <SheetHeader>
          <SheetTitle>New Document</SheetTitle>
          <SheetDescription>Give a name to your new document.</SheetDescription>
        </SheetHeader>
        <div className="flex flex-col gap-4 my-8">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            placeholder="Document Name"
            className="col-span-3"
            value={docName}
            onChange={(e) => setDocName(e.target.value)}
          />
        </div>
        <SheetFooter>
          <SheetClose asChild onClick={() => onDocumentCreate(docName)}>
            <Button type="submit">Create Document</Button>
          </SheetClose>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
