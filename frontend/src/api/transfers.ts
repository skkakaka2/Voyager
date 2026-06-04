import { CancelTransferTask, ListTransferTasks } from '../../wailsjs/go/main/App'
import type { TransferTask } from '../types/transfer'

export function listTransferTasks(): Promise<TransferTask[]> {
  return ListTransferTasks() as Promise<TransferTask[]>
}

export function cancelTransferTask(id: string): Promise<void> {
  return CancelTransferTask(id)
}
