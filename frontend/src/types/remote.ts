export type RemoteEntryType = 'file' | 'directory'

export interface RemoteEntry {
  name: string
  path: string
  type: RemoteEntryType
  size: number
  modifiedAt: string
}
