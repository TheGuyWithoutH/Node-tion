import { Skeleton } from "../ui/skeleton";

function DocumentLoader() {
  return (
    <div className="space-y-2 ml-12 pt-2">
      <Skeleton className="h-4 w-[70%] bg-gray-200" />
      <Skeleton className="h-4 w-[80%] bg-gray-200" />
      <Skeleton className="h-4 w-[60%] bg-gray-200" />
      <Skeleton className="h-4 w-[90%] bg-gray-200" />
      <Skeleton className="h-4 w-[40%] bg-gray-200" />
      <Skeleton className="h-4 w-[50%] bg-gray-200" />
      <Skeleton className="h-4 w-[70%] bg-gray-200" />
      <Skeleton className="h-4 w-[80%] bg-gray-200" />
    </div>
  );
}

export default DocumentLoader;
