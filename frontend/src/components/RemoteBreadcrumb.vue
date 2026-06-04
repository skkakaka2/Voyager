<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  path: string
}>()

defineEmits<{
  (event: 'navigate', value: string): void
}>()

const segments = computed(() => {
  const parts = props.path.split('/').filter(Boolean)
  return parts.map((label, index) => ({
    label,
    path: `/${parts.slice(0, index + 1).join('/')}`
  }))
})
</script>

<template>
  <el-breadcrumb separator="/">
    <el-breadcrumb-item>
      <button class="breadcrumb-button" type="button" @click="$emit('navigate', '/')">根目录</button>
    </el-breadcrumb-item>
    <el-breadcrumb-item v-for="segment in segments" :key="segment.path">
      <button class="breadcrumb-button" type="button" @click="$emit('navigate', segment.path)">
        {{ segment.label }}
      </button>
    </el-breadcrumb-item>
  </el-breadcrumb>
</template>

<style scoped>
.breadcrumb-button {
  border: 0;
  background: transparent;
  color: #409eff;
  font: inherit;
  cursor: pointer;
}
</style>
