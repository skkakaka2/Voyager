import { ref } from 'vue'
import { defineStore } from 'pinia'
import * as transferApi from '../api/transfers'
import type { TransferTask } from '../types/transfer'

export const useTransferStore = defineStore('transfers', () => {
  const items = ref<TransferTask[]>([])

  async function load() {
    items.value = await transferApi.listTransferTasks()
  }

  async function cancel(id: string) {
    await transferApi.cancelTransferTask(id)
    await load()
  }

  return { cancel, items, load }
})
