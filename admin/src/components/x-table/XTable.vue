<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   x-table 配置化 CRUD — 列表+分页+搜索+增改弹窗+删除+只读/行操作/列筛选排序
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  | @updated   2026-06-09 14:30:00  T-007c：只读模式 + 工具栏/行操作插槽 + 列筛选/排序（服务端）
  +----------------------------------------------------------------------
  说明：通用底座组件，业务中立；对接 T-007a 请求层（统一包络/错误码/hashid 透传）。
       内置增删改按钮挂 v-permission（permPrefix:create/update/delete），仅 UX，后端才是边界。
       新能力全走可选配置/插槽，缺省行为 = T-007b 现状（既有用户/角色页零回归）。
       筛选/排序仅前端拼参数透传后端，后端为过滤/排序权威；前端不做内存伪装。
-->
<script setup lang="ts">
import { ref, reactive, computed, onMounted, useSlots } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Search, RefreshLeft, Edit, Delete, Filter } from '@element-plus/icons-vue'
import type { XTableConfig, XField, XRow, XAction } from './types'

const props = defineProps<{ config: XTableConfig }>()
const { t } = useI18n()
const slots = useSlots()

const rowKey = computed(() => props.config.rowKey || 'id')
const permPrefix = computed(() => props.config.permPrefix || '')
const readonly = computed(() => props.config.readonly === true)

// ---- 列表 + 分页 ----
const loading = ref(false)
const rows = ref<XRow[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

// ---- 搜索栏 ----
const searchForm = reactive<Record<string, unknown>>({})
;(props.config.search || []).forEach((s) => (searchForm[s.prop] = ''))

// ---- 列筛选（表头入口）----
const columnFilters = reactive<Record<string, string>>({})
const filterPop = reactive<Record<string, boolean>>({})
props.config.columns.forEach((c) => {
  if (c.filterable) {
    columnFilters[c.prop] = ''
    filterPop[c.prop] = false
  }
})

// ---- 列排序（服务端）----
const sortState = reactive<{ prop: string; order: '' | 'asc' | 'desc' }>({ prop: '', order: '' })

async function fetchData(): Promise<void> {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    // 搜索栏
    for (const [k, v] of Object.entries(searchForm)) {
      if (v !== '' && v !== undefined && v !== null) params[k] = v
    }
    // 列筛选
    for (const [k, v] of Object.entries(columnFilters)) {
      if (v !== '' && v !== undefined && v !== null) params[k] = v
    }
    // 列排序
    if (sortState.order && sortState.prop) {
      params.sort = sortState.prop
      params.order = sortState.order
    }
    const res = await props.config.api.list(params)
    rows.value = res.list || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function onSearch(): void {
  page.value = 1
  fetchData()
}
function onReset(): void {
  for (const k of Object.keys(searchForm)) searchForm[k] = ''
  onSearch()
}
function onPageChange(p: number): void {
  page.value = p
  fetchData()
}

// 列筛选 apply/reset
function applyColumnFilter(prop: string): void {
  filterPop[prop] = false
  page.value = 1
  fetchData()
}
function resetColumnFilter(prop: string): void {
  columnFilters[prop] = ''
  filterPop[prop] = false
  page.value = 1
  fetchData()
}

// 列排序变化（el-table sort-change）→ 拼参数重新拉取
function onSortChange({ prop, order }: { prop: string; order: string | null }): void {
  if (order) {
    sortState.prop = prop
    sortState.order = order === 'ascending' ? 'asc' : 'desc'
  } else {
    sortState.prop = ''
    sortState.order = ''
  }
  page.value = 1
  fetchData()
}

// ---- 内置新增/编辑弹窗 ----
const fields = computed<XField[]>(() => props.config.fields || [])
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const form = reactive<Record<string, unknown>>({})
const editingId = ref('')

// 当前模式下可见字段（编辑时排除 createOnly）
const visibleFields = computed<XField[]>(() =>
  fields.value.filter((f) => !(dialogMode.value === 'edit' && f.createOnly)),
)

const rules = computed<FormRules>(() => {
  const r: FormRules = {}
  for (const f of visibleFields.value) {
    if (f.required) {
      r[f.prop] = [{ required: true, message: t('table.inputPrefix') + f.label, trigger: 'blur' }]
    }
  }
  return r
})

const dialogTitle = computed(() => (dialogMode.value === 'create' ? t('table.addTitle') : t('table.editTitle')))

function fieldDisabled(f: XField): boolean {
  return dialogMode.value === 'edit' && f.editable === false
}

function openCreate(): void {
  dialogMode.value = 'create'
  editingId.value = ''
  for (const f of fields.value) form[f.prop] = f.default ?? ''
  dialogVisible.value = true
}

function openEdit(row: XRow): void {
  dialogMode.value = 'edit'
  editingId.value = String(row[rowKey.value])
  for (const f of fields.value) form[f.prop] = row[f.prop] ?? f.default ?? ''
  dialogVisible.value = true
}

async function submitForm(): Promise<void> {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  // 仅提交当前可见字段
  const payload: Record<string, unknown> = {}
  for (const f of visibleFields.value) payload[f.prop] = form[f.prop]

  if (dialogMode.value === 'create') {
    await props.config.api.create?.(payload)
    ElMessage.success(t('table.addOk'))
  } else {
    await props.config.api.update?.(editingId.value, payload)
    ElMessage.success(t('table.saveOk'))
  }
  dialogVisible.value = false
  fetchData()
}

async function onDelete(row: XRow): Promise<void> {
  await ElMessageBox.confirm(t('table.delConfirm'), t('table.tip'), {
    confirmButtonText: t('table.confirm'),
    cancelButtonText: t('table.cancel'),
    type: 'warning',
  })
  await props.config.api.remove?.(String(row[rowKey.value]))
  ElMessage.success(t('table.delOk'))
  // 删除后若本页空了，回退一页
  if (rows.value.length === 1 && page.value > 1) page.value--
  fetchData()
}

// ---- 自定义行操作 ----
async function runAction(a: XAction, row: XRow): Promise<void> {
  if (a.confirm) {
    try {
      await ElMessageBox.confirm(a.confirm, t('table.tip'), {
        confirmButtonText: t('table.confirm'),
        cancelButtonText: t('table.cancel'),
        type: 'warning',
      })
    } catch {
      return // 用户取消
    }
  }
  await a.handler(row)
  if (a.refresh) fetchData()
}

// ---- 工具栏 / 操作列 是否渲染（缺省即 T-007b 现状）----
const showAddBtn = computed(() => !readonly.value && !!permPrefix.value)
const showToolbar = computed(() => showAddBtn.value || !!slots.toolbar)
const showOpColumn = computed(
  () => !!slots['row-actions'] || (props.config.actions?.length ?? 0) > 0 || !readonly.value,
)

onMounted(fetchData)
defineExpose({ refresh: fetchData })
</script>

<template>
  <div class="x-table">
    <!-- 搜索栏 -->
    <div v-if="config.search?.length" class="x-table__search">
      <el-form inline @submit.prevent>
        <el-form-item v-for="s in config.search" :key="s.prop" :label="s.label">
          <el-select
            v-if="s.type === 'select'"
            v-model="searchForm[s.prop]"
            :placeholder="s.placeholder || t('table.selectPlaceholder')"
            clearable
            style="width: 160px"
          >
            <el-option v-for="o in s.options" :key="String(o.value)" :label="o.label" :value="o.value" />
          </el-select>
          <el-input
            v-else
            v-model="searchForm[s.prop] as string"
            :placeholder="s.placeholder || t('table.inputPrefix') + s.label"
            clearable
            style="width: 180px"
            @keyup.enter="onSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="onSearch">{{ t('table.search') }}</el-button>
          <el-button :icon="RefreshLeft" @click="onReset">{{ t('table.reset') }}</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 工具栏：内置新增按钮 + 自定义插槽（不传且无新增按钮则整块不渲染、无空位） -->
    <div v-if="showToolbar" class="x-table__toolbar">
      <el-button
        v-if="showAddBtn"
        v-permission="`${permPrefix}:create`"
        type="primary"
        :icon="Plus"
        @click="openCreate"
      >
        {{ t('table.add') }}
      </el-button>
      <slot name="toolbar" />
    </div>

    <!-- 表格 -->
    <el-table
      v-loading="loading"
      :data="rows"
      border
      stripe
      :row-key="rowKey"
      @sort-change="onSortChange"
    >
      <el-table-column
        v-for="col in config.columns"
        :key="col.prop"
        :prop="col.prop"
        :label="col.label"
        :width="col.width"
        :min-width="col.minWidth"
        :sortable="col.sortable ? 'custom' : false"
        show-overflow-tooltip
      >
        <!-- 列筛选：表头入口（popover 输入 → 并入 query） -->
        <template v-if="col.filterable" #header>
          <span class="x-table__col-header">
            <span>{{ col.label }}</span>
            <el-popover v-model:visible="filterPop[col.prop]" :width="230" trigger="click" placement="bottom">
              <template #reference>
                <el-icon
                  class="x-table__filter-icon"
                  :class="{ 'is-active': !!columnFilters[col.prop] }"
                  @click.stop
                >
                  <Filter />
                </el-icon>
              </template>
              <div class="x-table__filter-pop" @click.stop>
                <el-input
                  v-model="columnFilters[col.prop]"
                  size="small"
                  clearable
                  :placeholder="t('table.inputPrefix') + col.label"
                  @keyup.enter="applyColumnFilter(col.prop)"
                />
                <div class="x-table__filter-btns">
                  <el-button size="small" text @click="resetColumnFilter(col.prop)">
                    {{ t('table.reset') }}
                  </el-button>
                  <el-button size="small" type="primary" @click="applyColumnFilter(col.prop)">
                    {{ t('table.confirm') }}
                  </el-button>
                </div>
              </div>
            </el-popover>
          </span>
        </template>

        <template v-if="col.formatter" #default="{ row }">
          {{ col.formatter(row, row[col.prop]) }}
        </template>
      </el-table-column>

      <el-table-column
        v-if="showOpColumn"
        :label="t('table.op')"
        :width="config.actionsWidth || 160"
        fixed="right"
      >
        <template #default="{ row }">
          <!-- 行操作插槽：完全自定义优先；缺省走内置 + 配置 actions -->
          <slot name="row-actions" :row="row">
            <el-button
              v-if="!readonly"
              v-permission="`${permPrefix}:update`"
              link
              type="primary"
              :icon="Edit"
              @click="openEdit(row)"
            >
              {{ t('table.edit') }}
            </el-button>
            <el-button
              v-if="!readonly"
              v-permission="`${permPrefix}:delete`"
              link
              type="danger"
              :icon="Delete"
              @click="onDelete(row)"
            >
              {{ t('table.del') }}
            </el-button>
            <!-- 配置行操作：有 perm 才挂 v-permission；无 perm = 公开按钮、不挂指令（非喂空值） -->
            <template v-for="a in config.actions || []" :key="a.label">
              <el-button
                v-if="a.perm"
                v-permission="a.perm"
                link
                :type="a.type || 'primary'"
                :icon="a.icon"
                @click="runAction(a, row)"
              >
                {{ a.label }}
              </el-button>
              <el-button
                v-else
                link
                :type="a.type || 'primary'"
                :icon="a.icon"
                @click="runAction(a, row)"
              >
                {{ a.label }}
              </el-button>
            </template>
          </slot>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <div class="x-table__pager">
      <el-pagination
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        background
        @current-change="onPageChange"
      />
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <el-form-item v-for="f in visibleFields" :key="f.prop" :label="f.label" :prop="f.prop">
          <el-select
            v-if="f.type === 'select'"
            v-model="form[f.prop]"
            :disabled="fieldDisabled(f)"
            style="width: 100%"
          >
            <el-option v-for="o in f.options" :key="String(o.value)" :label="o.label" :value="o.value" />
          </el-select>
          <el-input-number
            v-else-if="f.type === 'number'"
            v-model="form[f.prop] as number"
            :disabled="fieldDisabled(f)"
            controls-position="right"
          />
          <el-input
            v-else-if="f.type === 'textarea'"
            v-model="form[f.prop] as string"
            type="textarea"
            :disabled="fieldDisabled(f)"
            :placeholder="f.placeholder"
          />
          <el-input
            v-else
            v-model="form[f.prop] as string"
            :type="f.type === 'password' ? 'password' : 'text'"
            :show-password="f.type === 'password'"
            :disabled="fieldDisabled(f)"
            :placeholder="f.placeholder"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('table.cancel') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ t('table.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.x-table {
  &__search {
    margin-bottom: 8px;
  }
  &__toolbar {
    margin-bottom: 12px;
    display: flex;
    gap: 8px;
    align-items: center;
  }
  &__pager {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
  }
  &__col-header {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
  &__filter-icon {
    cursor: pointer;
    color: var(--el-text-color-secondary);
    &.is-active {
      color: var(--el-color-primary);
    }
  }
  &__filter-pop {
    .x-table__filter-btns {
      margin-top: 8px;
      display: flex;
      justify-content: flex-end;
      gap: 8px;
    }
  }
}
</style>
