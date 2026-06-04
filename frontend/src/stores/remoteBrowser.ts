import { ref } from 'vue'
import { defineStore } from 'pinia'
import * as remoteApi from '../api/remote'
import type { RemoteEntry } from '../types/remote'

export const useRemoteBrowserStore = defineStore('remoteBrowser', () => {
  const connectionId = ref('')
  const currentPath = ref('/')
  const entries = ref<RemoteEntry[]>([])
  const loading = ref(false)
  const selectedEntry = ref<RemoteEntry>()

  function reset() {
    connectionId.value = ''
    currentPath.value = '/'
    entries.value = []
    selectedEntry.value = undefined
  }

  function resetForConnection(id: string) {
    connectionId.value = id
    currentPath.value = '/'
    entries.value = []
    selectedEntry.value = undefined
  }

  async function load(path = currentPath.value) {
    if (!connectionId.value) {
      entries.value = []
      return
    }
    loading.value = true
    try {
      entries.value = await remoteApi.listFiles(connectionId.value, path)
      currentPath.value = path
      selectedEntry.value = undefined
    } finally {
      loading.value = false
    }
  }

  async function navigate(path: string) {
    currentPath.value = path
    selectedEntry.value = undefined
    await load(path)
  }

  async function createFolder(name: string) {
    if (!connectionId.value) {
      return
    }
    await remoteApi.createFolder(connectionId.value, currentPath.value, name)
    await load()
  }

  async function renameEntry(entry: RemoteEntry, newName: string) {
    if (!connectionId.value) {
      return
    }
    await remoteApi.renameEntry(connectionId.value, entry.path, newName)
    await load()
  }

  async function deleteEntry(entry: RemoteEntry) {
    if (!connectionId.value) {
      return
    }
    await remoteApi.deleteEntry(connectionId.value, entry.path)
    await load()
  }

  async function uploadFiles(localPaths: string[]) {
    if (!connectionId.value || localPaths.length === 0) {
      return []
    }
    const tasks = await remoteApi.uploadFiles(connectionId.value, currentPath.value, localPaths)
    await load()
    return tasks
  }

  async function downloadFile(entry: RemoteEntry, localDir: string) {
    if (!connectionId.value) {
      return undefined
    }
    return remoteApi.downloadFile(connectionId.value, entry.path, localDir, entry.size)
  }

  function selectEntry(entry?: RemoteEntry) {
    selectedEntry.value = entry
  }

  return {
    connectionId,
    createFolder,
    currentPath,
    deleteEntry,
    downloadFile,
    entries,
    load,
    loading,
    navigate,
    renameEntry,
    reset,
    resetForConnection,
    selectEntry,
    selectedEntry,
    uploadFiles
  }
})
