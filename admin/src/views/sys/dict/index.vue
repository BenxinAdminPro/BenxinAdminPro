<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   字典管理 — 类型↔数据双表联动（x-table 增强首个消费页：行操作 XAction + #toolbar + 列筛选）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-10 09:38:01
  | @updated   2026-06-14 11:45:00  T-005b-4：dict_data 改后端真分页（去本地切全量）
  +----------------------------------------------------------------------
  说明：左表字典类型（标准 CRUD），行操作「字典数据」选中类型 → 右表加载该类型数据（联动）。
       未选类型右侧给空态、不发空 dict_type 请求；切换类型经 :key 重建数据表刷新。
       T-005b-4：后端 /sys/dict/data 已支持真分页，适配层直接透传 page/page_size + dict_type。
       T-005b-4：后端 /sys/dict/types 已支持 dict_type 模糊 + created_at 排序，列筛选/排序真生效。
-->
<script setup lang="ts">
import { ref, computed } from 'vue'
import { Collection } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'
import {
  listDictTypes,
  createDictType,
  updateDictType,
  removeDictType,
  listDictData,
  createDictData,
  updateDictData,
  removeDictData,
  type DictTypeRow,
} from '@/api/dict'
import { statusText, dateText } from '@/utils/format'

const statusOptions = [
  { label: '正常', value: 0 },
  { label: '停用', value: 1 },
]

// ---- 主从联动锚点：当前选中的字典类型 ----
const selectedType = ref<DictTypeRow | null>(null)
// 切换类型 → :key 变化重建数据表（重新拉取并重置分页）
const dataTableKey = computed(() => selectedType.value?.dict_type ?? '')

// ---- 左表：字典类型 ----
const typeConfig: XTableConfig = {
  permPrefix: 'sys:dict',
  api: {
    list: listDictTypes,
    create: createDictType,
    update: updateDictType,
    // 删除选中类型时同步清空右侧数据区，避免悬空联动
    remove: async (id: string) => {
      await removeDictType(id)
      if (selectedType.value?.id === id) selectedType.value = null
    },
  },
  columns: [
    // T-005b-4：dict_type 模糊过滤 + created_at 排序后端已就绪，filterable/sortable 真生效
    { prop: 'dict_type', label: '字典类型', minWidth: 150, filterable: true },
    { prop: 'name', label: '名称', minWidth: 110 },
    { prop: 'status', label: '状态', width: 80, formatter: (_r, v) => statusText(v) },
    { prop: 'created_at', label: '创建时间', minWidth: 165, sortable: true, formatter: (_r, v) => dateText(v) },
  ],
  fields: [
    // dict_type 为唯一键：编辑禁改（避免后端无级联、改名孤儿化既有数据）
    { prop: 'dict_type', label: '字典类型', required: true, editable: false, placeholder: '如 sys_user_status' },
    { prop: 'name', label: '名称', required: true },
    { prop: 'status', label: '状态', type: 'select', default: 0, options: statusOptions },
    { prop: 'remark', label: '备注', type: 'textarea' },
  ],
  actions: [
    {
      label: '字典数据',
      perm: 'sys:dict:list',
      icon: Collection,
      handler: (row: XRow) => {
        selectedType.value = row as DictTypeRow
      },
    },
  ],
  actionsWidth: 240,
}

// ---- 右表：字典数据（随选中类型联动；dict_type 由适配层注入，表单不暴露）----
const dataConfig: XTableConfig = {
  permPrefix: 'sys:dict',
  api: {
    list: async (params: Record<string, unknown>) => {
      // T-005b-4：后端真分页，直接透传 page/page_size + 选中 dict_type（不再本地切全量）。
      const dictType = selectedType.value?.dict_type ?? ''
      if (!dictType) return { list: [], total: 0, page: 1, page_size: 10 }
      return await listDictData({ ...params, dict_type: dictType })
    },
    create: (data) => createDictData({ ...data, dict_type: selectedType.value?.dict_type }),
    update: (id, data) => updateDictData(id, { ...data, dict_type: selectedType.value?.dict_type }),
    remove: removeDictData,
  },
  columns: [
    { prop: 'label', label: '标签', minWidth: 100 },
    { prop: 'value', label: '键值', minWidth: 100 },
    { prop: 'sort', label: '排序', width: 70 },
    { prop: 'status', label: '状态', width: 80, formatter: (_r, v) => statusText(v) },
  ],
  fields: [
    { prop: 'label', label: '标签', required: true },
    { prop: 'value', label: '键值', required: true },
    { prop: 'sort', label: '排序', type: 'number', default: 0 },
    { prop: 'status', label: '状态', type: 'select', default: 0, options: statusOptions },
  ],
}
</script>

<template>
  <el-row :gutter="12">
    <el-col :xs="24" :md="14">
      <el-card shadow="never">
        <template #header>字典类型</template>
        <XTable :config="typeConfig" />
      </el-card>
    </el-col>
    <el-col :xs="24" :md="10">
      <el-card shadow="never">
        <template #header>字典数据</template>
        <XTable v-if="selectedType" :key="dataTableKey" :config="dataConfig">
          <!-- 工具栏插槽：标示当前联动的类型上下文 -->
          <template #toolbar>
            <el-tag type="info">
              当前类型：{{ selectedType.name }}（{{ selectedType.dict_type }}）
            </el-tag>
          </template>
        </XTable>
        <el-empty v-else description="请在左侧点击「字典数据」选择一个字典类型" />
      </el-card>
    </el-col>
  </el-row>
</template>
