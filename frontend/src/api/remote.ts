import {
  CreateFolder,
  DeleteEntry,
  DownloadFile,
  ListFiles,
  PickDownloadDirectory,
  PickUploadFiles,
  RenameEntry,
  UploadFiles
} from '../../wailsjs/go/main/App'
import type { RemoteEntry } from '../types/remote'
import type { TransferTask } from '../types/transfer'

export function listFiles(connectionId: string, path: string): Promise<RemoteEntry[]> {
  return ListFiles(connectionId, path) as Promise<RemoteEntry[]>
}

export function createFolder(connectionId: string, parentPath: string, name: string): Promise<void> {
  return CreateFolder(connectionId, parentPath, name)
}

export function renameEntry(connectionId: string, oldPath: string, newName: string): Promise<void> {
  return RenameEntry(connectionId, oldPath, newName)
}

export function deleteEntry(connectionId: string, path: string): Promise<void> {
  return DeleteEntry(connectionId, path)
}

export function pickUploadFiles(): Promise<string[]> {
  return PickUploadFiles()
}

export function uploadFiles(connectionId: string, remoteDir: string, localPaths: string[]): Promise<TransferTask[]> {
  return UploadFiles(connectionId, remoteDir, localPaths) as Promise<TransferTask[]>
}

export function pickDownloadDirectory(): Promise<string> {
  return PickDownloadDirectory()
}

export function downloadFile(
  connectionId: string,
  remotePath: string,
  localDir: string,
  bytesTotal: number
): Promise<TransferTask> {
  return DownloadFile(connectionId, remotePath, localDir, bytesTotal) as Promise<TransferTask>
}
