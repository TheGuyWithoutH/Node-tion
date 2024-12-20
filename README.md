<img src="https://github.com/user-attachments/assets/0f56020c-70d6-4028-9d12-2303988bdf4b" width="150">

# BalduchColab - A Decentralized Block Editor App Based on Peritext



## Introduction

BalduchColab is a decentralized rich text block editor based on [Peritext](https://www.inkandswitch.com/peritext/) CRDT algorithm, built in Go and React. It uses a Peerster network to share document updates among peers in the network. Our objective is to make a user friendly application that can be installed to try out a Peritext like algorithm that includes blocks.

## Installation

For this project, you need NodeJS (preferrably v18.17.0), Go (v1.23.1). For more instructions on how to install them, please refer to the dedicated webpages for [Node](https://nodejs.org/en/download/package-manager/current) and [Go](https://go.dev/doc/install).
You need to install the following libraries:

### Go Libraries

For the desktop app framework, you need to install Wails:

```console
go go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

We also use another library to get the IP Adress of the host:

```console
go go install github.com/jackpal/gateway
```

### Frontend Libraries

To install all the javascript dependencies, you can run the following commands:

```console
cd frontend
npm install
```

## Live Preview

To run in live preview mode, run `wails dev` in the project directory. Wails will compile the application and start a window with the editor.

## Building

To build a redistributable, production mode package, use `wails build`.

## Trying the App

### Add a Peer

To use BalduchColab with someone else, you need to first make sure that all the computers are in the same local network. You can then all launch the app and go to the 'AddPeer' screen.
Here, you find the node IP address and port to share with other peers. You have to add each other using those addresses and ports. Once this is done, you can go on to editing a page in the Editor.

<p align="center">
<img width="500" alt="image" src="https://github.com/user-attachments/assets/1f40ed61-8b47-4486-9e55-f4a17540f5ac" />
</p>


### Create a Document

To create a document, click on the 'New Document' and give it a name. All the peers in the network that will participate in the editing of the document will have to do the same procedure. After that, you can all edit the page with the block editor.

<p align="center">
<img width="500" alt="image" src="https://github.com/user-attachments/assets/0d317ba4-c35f-48de-ab81-fcfafea75605" />
</p>


### Syncing a Document

At the top of the editor, you see a 'Sync' button. This allows you to:
1. Send your changes to all peers in the network
2. Get the new updated document given the modifications from everyone

<p align="center">
<img width="500" alt="image" src="https://github.com/user-attachments/assets/8e7f5a71-ee31-4c5a-9d2a-d46f8b68d64f" />
</p>

### Disclaimer

The implementation still needs to integrate some block types. Here are the currently not supported types:
- Code Block
- Table
- Image
- Audio
- Video
- File
- Emoji

We also need to refine the behavior detected in certain edge cases. If you find any bug, do not hesitate to submit an Issue to the repo: [Create Issue](https://github.com/cs438-epfl/2024-proj-balduchcolab/issues/new)




