import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as connectionApi from '../api/connections'
import type { ConnectionInput, ConnectionProfile } from '../types/connection'

export const useConnectionStore = defineStore('connections', () => {
  const items = ref<ConnectionProfile[]>([])
  const currentId = ref('')
  const loading = ref(false)
  const saving = ref(false)

  const currentConnection = computed(() => items.value.find((item) => item.id === currentId.value))

  async function load() {
    loading.value = true
    try {
      items.value = await connectionApi.listConnections()
      if (!currentId.value && items.value.length > 0) {
        currentId.value = items.value[0].id
      }
    } finally {
      loading.value = false
    }
  }

  async function save(input: ConnectionInput) {
    saving.value = true
    try {
      const profile = await connectionApi.saveConnection(input)
      const index = items.value.findIndex((item) => item.id === profile.id)
      if (index >= 0) {
        items.value[index] = profile
      } else {
        items.value.unshift(profile)
      }
      currentId.value = profile.id
      return profile
    } finally {
      saving.value = false
    }
  }

  async function remove(id: string) {
    await connectionApi.deleteConnection(id)
    items.value = items.value.filter((item) => item.id !== id)
    if (currentId.value === id) {
      currentId.value = items.value[0]?.id ?? ''
    }
  }

  async function test(connection: ConnectionProfile) {
    await connectionApi.testConnection({
      id: connection.id,
      name: connection.name,
      protocol: connection.protocol,
      host: connection.host,
      port: connection.port,
      baseUrl: connection.baseUrl,
      share: connection.share,
      rootPath: connection.rootPath,
      username: connection.username,
      domain: connection.domain,
      passiveMode: connection.passiveMode,
      savePassword: connection.passwordSaved
    })
  }

  function select(id: string) {
    currentId.value = id
  }

  return { currentConnection, currentId, items, loading, load, remove, save, saving, select, test }
})
