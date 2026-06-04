<script setup lang="ts">
import { Delete, Download, FolderAdd, Refresh, Upload } from '@element-plus/icons-vue'
import type { RemoteEntry } from '../types/remote'

defineProps<{
  hasConnection: boolean
  selectedEntry?: RemoteEntry
}>()

defineEmits<{
  (event: 'refresh'): void
  (event: 'upload'): void
  (event: 'download'): void
  (event: 'mkdir'): void
  (event: 'rename'): void
  (event: 'remove'): void
}>()
</script>

<template>
  <div class="file-toolbar">
    <div class="file-toolbar__left">
      <el-tooltip content="刷新" placement="bottom">
        <el-button :disabled="!hasConnection" :icon="Refresh" circle @click="$emit('refresh')" />
      </el-tooltip>
      <el-tooltip content="上传" placement="bottom">
        <el-button :disabled="!hasConnection" :icon="Upload" circle @click="$emit('upload')" />
      </el-tooltip>
      <el-tooltip content="下载" placement="bottom">
        <el-button :disabled="!selectedEntry || selectedEntry.type === 'directory'" :icon="Download" circle @click="$emit('download')" />
      </el-tooltip>
      <el-tooltip content="新建文件夹" placement="bottom">
        <el-button :disabled="!hasConnection" :icon="FolderAdd" circle @click="$emit('mkdir')" />
      </el-tooltip>
    </div>

    <div class="file-toolbar__right">
      <el-button :disabled="!selectedEntry" @click="$emit('rename')">重命名</el-button>
      <el-button :disabled="!selectedEntry" :icon="Delete" type="danger" @click="$emit('remove')">删除</el-button>
    </div>
  </div>
</template>

<style scoped>
.file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  background: #ffffff;
  padding: 10px 12px;
}

.file-toolbar__left,
.file-toolbar__right {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
