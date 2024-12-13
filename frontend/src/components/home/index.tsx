import icon from "../../assets/icon.png";

/**
 * Home component that explains the purpose of the application and how to use it.
 * Using shadcn's components and libraries.
 */
export default function Home() {
  return (
    <div className="flex flex-col items-center justify-center h-full">
      <img
        src={icon}
        alt="Node-tion logo"
        className="w-32 h-32 no-select pointer-events-none"
      />
      <h1 className="text-4xl font-bold py-4">Welcome to BalduchColab</h1>
      <p className="text-lg mt-4">
        BalduchColab is a note-taking app that allows you to create and manage
        notes using blocks. The app is totally decentralized thanks to the
        Peritext protocol.
      </p>
      <p className="text-lg mt-4">
        Peritext is an algorithm for rich-text collaboration that provides
        greater flexibility: it allows users to edit independent copies of a
        document, and it provides a mechanism for automatically merging those
        versions back together in a way that preserves the users’ intent as much
        as possible. Once the versions are merged, the algorithm guarantees that
        all users converge towards the same merged result.
      </p>

      <a
        href="https://www.inkandswitch.com/peritext/"
        target="_blank"
        className="text-blue-500 my-4"
      >
        Learn more about Peritext
      </a>
      <p className="text-lg mt-4">
        To get started, click on the editor tab in the sidebar to create a new
        note.
      </p>
      <p className="text-sm mt-4 text-gray-500 mt-16">
        This project has been developed by Yasmin Ben Rahhal, Emma Gaia Poggiolini, Emile Hreich and Ugo Balducci in the
        context of the course CS-438 at EPFL.
      </p>
    </div>
  );
}
