<script setup lang="ts">
import type { RemoteEntry } from '../types/remote'

const props = defineProps<{
  entries: RemoteEntry[]
  loading: boolean
  selectedPath?: string
}>()

const emit = defineEmits<{
  (event: 'select', value?: RemoteEntry): void
  (event: 'open', value: RemoteEntry): void
}>()

function formatSize(row: RemoteEntry) {
  if (row.type === 'directory') {
    return '-'
  }
  if (row.size < 1024) {
    return `${row.size} B`
  }
  if (row.size < 1024 * 1024) {
    return `${(row.size / 1024).toFixed(1)} KB`
  }
  return `${(row.size / 1024 / 1024).toFixed(1)} MB`
}

function rowClassName({ row }: { row: RemoteEntry }) {
  return row.path === props.selectedPath ? 'is-selected' : ''
}

function rowKey(row: RemoteEntry) {
  return row.path
}

function selectCurrent(row?: RemoteEntry) {
  emit('select', row)
}

function openRow(row: RemoteEntry) {
  emit('open', row)
}
</script>

<template>
  <el-table
    class="remote-table"
    :data="entries"
    :highlight-current-row="true"
    :row-class-name="rowClassName"
    :row-key="rowKey"
    v-loading="loading"
    empty-text="当前目录暂无文件"
    @current-change="selectCurrent"
    @row-dblclick="openRow"
  >
    <el-table-column label="名称" min-width="260" prop="name" />
    <el-table-column label="类型" width="120">
      <template #default="{ row }">
        <el-tag :type="row.type === 'directory' ? 'primary' : 'info'" size="small">
          {{ row.type === 'directory' ? '文件夹' : '文件' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="大小" width="140">
      <template #default="{ row }">{{ formatSize(row) }}</template>
    </el-table-column>
    <el-table-column label="修改时间" min-width="180" prop="modifiedAt" />
  </el-table>
</template>

<style scoped>
.remote-table {
  flex: 1;
  min-height: 360px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
}
</style>
