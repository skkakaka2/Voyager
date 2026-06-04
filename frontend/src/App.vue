<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ConnectionDialog from './components/ConnectionDialog.vue'
import ConnectionList from './components/ConnectionList.vue'
import EmptyState from './components/EmptyState.vue'
import FileToolbar from './components/FileToolbar.vue'
import RemoteBreadcrumb from './components/RemoteBreadcrumb.vue'
import RemoteFileTable from './components/RemoteFileTable.vue'
import TransferPanel from './components/TransferPanel.vue'
import * as remoteApi from './api/remote'
import { useConnectionStore } from './stores/connections'
import { useRemoteBrowserStore } from './stores/remoteBrowser'
import { useTransferStore } from './stores/transfers'
import type { ConnectionInput, ConnectionProfile } from './types/connection'
import type { RemoteEntry } from './types/remote'

const connections = useConnectionStore()
const remote = useRemoteBrowserStore()
const transfers = useTransferStore()

const dialogVisible = ref(false)
const editingConnection = ref<ConnectionProfile | null>(null)
let transferPollTimer: number | undefined

onMounted(async () => {
  try {
    await connections.load()
    if (connections.currentId) {
      remote.resetForConnection(connections.currentId)
      await remote.load('/')
      await transfers.load()
      if (hasActiveTransfer()) {
        startTransferPolling()
      }
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '连接列表加载失败')
  }
})

onUnmounted(() => {
  stopTransferPolling()
})

function openCreateDialog() {
  editingConnection.value = null
  dialogVisible.value = true
}

function openEditDialog(connection: ConnectionProfile) {
  editingConnection.value = connection
  dialogVisible.value = true
}

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === 'string' && error) {
    return error
  }
  return fallback
}

function hasActiveTransfer() {
  return transfers.items.some((task) => task.status === 'queued' || task.status === 'running')
}

function startTransferPolling() {
  if (transferPollTimer !== undefined) {
    return
  }
  transferPollTimer = window.setInterval(async () => {
    try {
      const hadActive = hasActiveTransfer()
      await transfers.load()
      if (hadActive && !hasActiveTransfer()) {
        await remote.load()
      }
      if (!hasActiveTransfer()) {
        stopTransferPolling()
      }
    } catch (error) {
      stopTransferPolling()
      ElMessage.error(errorMessage(error, '传输任务刷新失败'))
    }
  }, 1000)
}

function stopTransferPolling() {
  if (transferPollTimer === undefined) {
    return
  }
  window.clearInterval(transferPollTimer)
  transferPollTimer = undefined
}

async function saveConnection(input: ConnectionInput) {
  try {
    const profile = await connections.save(input)
    dialogVisible.value = false
    if (input.password && input.savePassword && !profile.passwordSaved) {
      ElMessage.warning('当前系统钥匙串不可用，密码不会被保存')
    } else {
      ElMessage.success('连接已保存')
    }
  } catch (error) {
    ElMessage.error(errorMessage(error, '连接保存失败'))
  }
}

async function removeConnection(connection: ConnectionProfile) {
  try {
    await ElMessageBox.confirm(`确定删除连接“${connection.name}”吗？`, '删除连接', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await connections.remove(connection.id)
    remote.reset()
    ElMessage.success('连接已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(errorMessage(error, '连接删除失败'))
    }
  }
}

async function testConnection(connection: ConnectionProfile) {
  try {
    await connections.test(connection)
    ElMessage.success('连接测试成功')
  } catch (error) {
    ElMessage.warning(errorMessage(error, '协议适配器尚未接入'))
  }
}

async function selectConnection(id: string) {
  connections.select(id)
  remote.resetForConnection(id)
  try {
    await remote.load('/')
  } catch (error) {
    ElMessage.error(errorMessage(error, '目录加载失败'))
  }
}

async function refreshCurrentPath() {
  try {
    await remote.load()
  } catch (error) {
    ElMessage.error(errorMessage(error, '目录刷新失败'))
  }
}

async function createFolder() {
  try {
    const result = await ElMessageBox.prompt('请输入文件夹名称', '新建文件夹', {
      confirmButtonText: '创建',
      cancelButtonText: '取消',
      inputPattern: /^(?!\s*$)[^/\\]+$/,
      inputErrorMessage: '名称不能为空，且不能包含路径分隔符'
    })
    await remote.createFolder(result.value.trim())
    ElMessage.success('文件夹已创建')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(errorMessage(error, '新建文件夹失败'))
    }
  }
}

async function renameSelectedEntry() {
  if (!remote.selectedEntry) {
    return
  }
  try {
    const result = await ElMessageBox.prompt('请输入新名称', '重命名', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputValue: remote.selectedEntry.name,
      inputPattern: /^(?!\s*$)[^/\\]+$/,
      inputErrorMessage: '名称不能为空，且不能包含路径分隔符'
    })
    await remote.renameEntry(remote.selectedEntry, result.value.trim())
    ElMessage.success('已重命名')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(errorMessage(error, '重命名失败'))
    }
  }
}

async function deleteSelectedEntry() {
  if (!remote.selectedEntry) {
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除“${remote.selectedEntry.name}”吗？`, '删除条目', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await remote.deleteEntry(remote.selectedEntry)
    ElMessage.success('已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(errorMessage(error, '删除失败'))
    }
  }
}

async function uploadFiles() {
  try {
    const localPaths = await remoteApi.pickUploadFiles()
    if (localPaths.length === 0) {
      return
    }
    const remoteNames = new Set(remote.entries.map((entry) => entry.name))
    const conflicts = Array.from(
      new Set(
        localPaths
          .map((localPath) => localPath.split(/[\\/]/).filter(Boolean).pop() ?? localPath)
          .filter((name) => remoteNames.has(name))
      )
    )
    if (conflicts.length > 0) {
      const conflictSummary =
        conflicts.length > 3
          ? `${conflicts.slice(0, 3).join('、')} 等 ${conflicts.length} 项`
          : conflicts.join('、')
      await ElMessageBox.confirm(
        `当前目录已存在同名条目：${conflictSummary}。继续上传将覆盖这些条目，确定继续吗？`,
        '覆盖确认',
        {
          confirmButtonText: '继续上传',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    }
    await remote.uploadFiles(localPaths)
    await transfers.load()
    startTransferPolling()
    ElMessage.success('上传任务已开始')
  } catch (error) {
    if (error === 'cancel' || error === 'close') {
      return
    }
    await transfers.load()
    ElMessage.error(errorMessage(error, '上传失败'))
  }
}

async function downloadSelectedEntry() {
  if (!remote.selectedEntry) {
    ElMessage.warning('请先选择要下载的文件')
    return
  }
  if (remote.selectedEntry.type === 'directory') {
    ElMessage.warning('暂不支持下载文件夹，请选择文件')
    return
  }
  try {
    const localDir = await remoteApi.pickDownloadDirectory()
    if (!localDir) {
      return
    }
    await remote.downloadFile(remote.selectedEntry, localDir)
    await transfers.load()
    startTransferPolling()
    ElMessage.success('下载任务已开始')
  } catch (error) {
    await transfers.load()
    ElMessage.error(errorMessage(error, '下载失败'))
  }
}

async function cancelTransfer(id: string) {
  try {
    await transfers.cancel(id)
    startTransferPolling()
    ElMessage.success('已取消传输任务')
  } catch (error) {
    ElMessage.error(errorMessage(error, '取消传输失败'))
  }
}

async function navigateRemote(path: string) {
  try {
    await remote.navigate(path)
  } catch (error) {
    ElMessage.error(errorMessage(error, '目录加载失败'))
  }
}

async function openEntry(entry: RemoteEntry) {
  if (entry.type !== 'directory') {
    return
  }
  await navigateRemote(entry.path)
}
</script>

<template>
  <el-container class="app-shell">
    <el-aside class="sidebar" width="300px">
      <ConnectionList
        :connections="connections.items"
        :current-id="connections.currentId"
        :loading="connections.loading"
        @create="openCreateDialog"
        @edit="openEditDialog"
        @remove="removeConnection"
        @select="selectConnection"
        @test="testConnection"
      />
    </el-aside>

    <el-container>
      <el-header class="workspace-header" height="56px">
        <RemoteBreadcrumb :path="remote.currentPath" @navigate="navigateRemote" />
      </el-header>

      <el-main class="workspace-main">
        <FileToolbar
          :has-connection="Boolean(connections.currentConnection)"
          :selected-entry="remote.selectedEntry"
          @download="downloadSelectedEntry"
          @mkdir="createFolder"
          @refresh="refreshCurrentPath"
          @remove="deleteSelectedEntry"
          @rename="renameSelectedEntry"
          @upload="uploadFiles"
        />

        <RemoteFileTable
          v-if="connections.currentConnection"
          :entries="remote.entries"
          :loading="remote.loading"
          :selected-path="remote.selectedEntry?.path"
          @open="openEntry"
          @select="remote.selectEntry"
        />
        <EmptyState
          v-else
          title="请选择一个连接"
          description="左侧保存远端服务连接后，可以在这里浏览目录和执行文件操作。"
        />
      </el-main>

      <el-footer class="transfer-footer" height="220px">
        <TransferPanel :tasks="transfers.items" @cancel="cancelTransfer" />
      </el-footer>
    </el-container>

    <ConnectionDialog
      v-model="dialogVisible"
      :connection="editingConnection"
      :submitting="connections.saving"
      @save="saveConnection"
    />
  </el-container>
</template>

<style scoped>
.app-shell {
  height: 100vh;
  min-width: 960px;
  background: #f5f7fa;
}

.sidebar {
  border-right: 1px solid #dcdfe6;
  background: #ffffff;
}

.workspace-header {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #dcdfe6;
  background: #ffffff;
}

.workspace-main {
  display: flex;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
}

.transfer-footer {
  border-top: 1px solid #dcdfe6;
  background: #ffffff;
  padding: 12px 16px;
}
</style>
