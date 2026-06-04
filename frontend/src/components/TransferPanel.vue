<script setup lang="ts">
import { Close } from '@element-plus/icons-vue'
import type { TransferTask } from '../types/transfer'

defineProps<{
  tasks: TransferTask[]
}>()

const emit = defineEmits<{
  (event: 'cancel', id: string): void
}>()

const directionLabels = {
  upload: '上传',
  download: '下载'
} as const

const statusLabels = {
  queued: '等待中',
  running: '进行中',
  succeeded: '成功',
  failed: '失败',
  canceled: '已取消'
} as const

const statusTypes = {
  queued: 'info',
  running: 'primary',
  succeeded: 'success',
  failed: 'danger',
  canceled: 'warning'
} as const

function directionLabel(task: TransferTask) {
  return directionLabels[task.direction]
}

function statusLabel(task: TransferTask) {
  return statusLabels[task.status]
}

function statusType(task: TransferTask) {
  return statusTypes[task.status]
}

function formatTime(value?: string) {
  if (!value) {
    return '-'
  }
  return new Date(value).toLocaleString()
}

function progressPercentage(task: TransferTask) {
  if (task.bytesTotal <= 0) {
    return task.status === 'succeeded' ? 100 : 0
  }
  return Math.min(100, Math.round((task.bytesDone / task.bytesTotal) * 100))
}

function formatBytes(value: number) {
  if (value <= 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = value
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  if (unitIndex === 0) {
    return `${Math.round(size)} ${units[unitIndex]}`
  }
  return `${size >= 10 ? size.toFixed(1) : size.toFixed(2)} ${units[unitIndex]}`
}

function progressText(task: TransferTask) {
  if (task.bytesTotal <= 0) {
    return formatBytes(task.bytesDone)
  }
  return `${formatBytes(task.bytesDone)} / ${formatBytes(task.bytesTotal)}`
}

function elapsedSeconds(task: TransferTask) {
  if (!task.startedAt) {
    return 0
  }
  const startedAt = new Date(task.startedAt).getTime()
  const endedAt = task.finishedAt ? new Date(task.finishedAt).getTime() : Date.now()
  return Math.max((endedAt - startedAt) / 1000, 0)
}

function bytesPerSecond(task: TransferTask) {
  const seconds = elapsedSeconds(task)
  if (seconds <= 0 || task.bytesDone <= 0) {
    return 0
  }
  return task.bytesDone / seconds
}

function formatSpeed(task: TransferTask) {
  const speed = bytesPerSecond(task)
  if (speed <= 0) {
    return '-'
  }
  return `${formatBytes(speed)}/s`
}

function formatDuration(totalSeconds: number) {
  if (totalSeconds < 1) {
    return '即将完成'
  }
  const seconds = Math.ceil(totalSeconds)
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const restSeconds = seconds % 60
  if (hours > 0) {
    return `${hours}小时${minutes}分`
  }
  if (minutes > 0) {
    return `${minutes}分${restSeconds}秒`
  }
  return `${restSeconds}秒`
}

function formatEta(task: TransferTask) {
  if (!canCancel(task)) {
    return '-'
  }
  if (task.bytesTotal <= 0 || task.bytesDone <= 0) {
    return '估算中'
  }
  const speed = bytesPerSecond(task)
  if (speed <= 0) {
    return '估算中'
  }
  return formatDuration((task.bytesTotal - task.bytesDone) / speed)
}

function canCancel(task: TransferTask) {
  return task.status === 'queued' || task.status === 'running'
}
</script>

<template>
  <section class="transfer-panel">
    <div class="transfer-panel__title">传输任务</div>
    <el-table :data="tasks" empty-text="暂无传输任务" height="160" size="small">
      <el-table-column label="方向" width="88">
        <template #default="{ row }">
          {{ directionLabel(row) }}
        </template>
      </el-table-column>
      <el-table-column label="来源" min-width="180">
        <template #default="{ row }">
          <span class="path-text" :title="row.source">{{ row.source }}</span>
        </template>
      </el-table-column>
      <el-table-column label="目标" min-width="180">
        <template #default="{ row }">
          <span class="path-text" :title="row.destination">{{ row.destination }}</span>
        </template>
      </el-table-column>
      <el-table-column label="进度" width="176">
        <template #default="{ row }">
          <div class="progress-cell">
            <el-progress :percentage="progressPercentage(row)" :show-text="false" />
            <span class="progress-cell__text">{{ progressText(row) }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="速度" width="110">
        <template #default="{ row }">
          {{ formatSpeed(row) }}
        </template>
      </el-table-column>
      <el-table-column label="剩余" width="104">
        <template #default="{ row }">
          {{ formatEta(row) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tooltip v-if="row.errorMessage" :content="row.errorMessage" placement="top">
            <el-tag :type="statusType(row)">{{ statusLabel(row) }}</el-tag>
          </el-tooltip>
          <el-tag v-else :type="statusType(row)">{{ statusLabel(row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="完成时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.finishedAt ?? row.startedAt ?? row.createdAt) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="64" align="center">
        <template #default="{ row }">
          <el-tooltip v-if="canCancel(row)" content="取消传输" placement="top">
            <el-button
              :icon="Close"
              aria-label="取消传输"
              circle
              size="small"
              type="danger"
              @click="emit('cancel', row.id)"
            />
          </el-tooltip>
          <span v-else class="operation-placeholder">-</span>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>

<style scoped>
.transfer-panel {
  display: flex;
  height: 100%;
  flex-direction: column;
  gap: 8px;
}

.transfer-panel__title {
  color: #303133;
  font-size: 14px;
  font-weight: 650;
}

.path-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.progress-cell__text,
.operation-placeholder {
  color: #909399;
  font-size: 12px;
}
</style>
