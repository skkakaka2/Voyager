export type TransferDirection = 'upload' | 'download'
export type TransferStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'

export interface TransferTask {
  id: string
  connectionId: string
  direction: TransferDirection
  source: string
  destination: string
  status: TransferStatus
  bytesDone: number
  bytesTotal: number
  errorMessage?: string
  createdAt: string
  startedAt?: string
  finishedAt?: string
}
