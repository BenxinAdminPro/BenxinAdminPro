<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   菜单管理 — parent_id 自引用树（el-table 内建 tree）+ M/C/F 三类型动态表单 CRUD
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-12 10:51:19
  +----------------------------------------------------------------------
  说明：后端 GET /sys/menus/tree 返回【已嵌套树】（服务端 buildMenuTree，sort ASC, id ASC），
       表格直接消费后端树（零再组装、不引入顺序分歧）；utils/tree.ts 的 buildTree 用于
       父级选择器——编辑时 flatten → 排除自己及子孙 → 重建（前端先挡"挂到自己子孙下"，
       后端 ancestors 校验仍是权威）。
       menu_type 约束以后端 validateMenuType 为准：F 必填 perm_code（且全局唯一）、
       M/C 禁填 perm_code；父子类型【后端无约束】，前端选择器如实镜像不擅自加限制。
       后端树接口无任何查询参数（search/filter/sort 均无）——本页不伪造前端假搜索，诚实降级。
       update 为全量覆写：表单带全字段提交；类型切换时 perm_code 按后端规则强制（M/C 提交空串），
       其余隐藏字段保留原值不清空（隐藏 ≠ 清除，最小破坏）。
-->
<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Sort } from '@element-plus/icons-vue'
import { getMenuTree, createMenu, updateMenu, removeMenu, type MenuRow, type MenuPayload } from '@/api/menu'
import { buildTree, flattenTree, subtreeIds } from '@/utils/tree'

// ---- 树数据（直接消费后端嵌套树）----
const loading = ref(false)
const tree = ref<MenuRow[]>([])

async function fetchTree(): Promise<void> {
  loading.value = true
  try {
    tree.value = (await getMenuTree()) || []
  } catch {
    // 请求层已按错误码 toast；保留旧数据
  } finally {
    loading.value = false
  }
}

// ---- 展开/折叠全部（经 :key 重建表格生效）----
const expandAll = ref(true)
const tableKey = computed(() => (expandAll.value ? 'expand' : 'collapse'))

// ---- 类型/状态展示 ----
const typeMeta: Record<string, { label: string; tag: 'warning' | 'primary' | 'info' }> = {
  M: { label: '目录', tag: 'warning' },
  C: { label: '菜单', tag: 'primary' },
  F: { label: '按钮', tag: 'info' },
}

// ---- 弹窗表单 ----
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref('')
const formRef = ref<FormInstance>()
const form = reactive({
  parent_id: '' as string,
  menu_type: 'C' as 'M' | 'C' | 'F',
  name: '',
  perm_code: '',
  path: '',
  component: '',
  icon: '',
  sort: 0,
  visible: 1,
  status: 0,
})

const dialogTitle = computed(() => (dialogMode.value === 'create' ? '新增菜单' : '编辑菜单'))

// 字段显隐按 menu_type（后端语义：F=权限点无路由；M=目录无组件；perm_code 仅 F）
const showPermCode = computed(() => form.menu_type === 'F')
const showPath = computed(() => form.menu_type !== 'F')
const showComponent = computed(() => form.menu_type === 'C')
const showIcon = computed(() => form.menu_type !== 'F')
const showVisible = computed(() => form.menu_type !== 'F')

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  ...(showPermCode.value
    ? { perm_code: [{ required: true, message: '按钮权限码必填（后端校验）', trigger: 'blur' }] }
    : {}),
}))

// 父级选择器选项：编辑时排除"自己及子孙"（防挂到自己子树下；后端 ancestors 校验仍是权威）。
// buildTree 真消费点：flatten 后过滤重建（T-007h dept 选择器复用同一工具）。
const parentOptions = computed<MenuRow[]>(() => {
  if (dialogMode.value !== 'edit' || !editingId.value) return tree.value
  const excluded = subtreeIds(tree.value, editingId.value)
  const flat = flattenTree(tree.value).filter((n) => !excluded.has(n.id))
  return buildTree(flat)
})

function resetForm(): void {
  form.parent_id = ''
  form.menu_type = 'C'
  form.name = ''
  form.perm_code = ''
  form.path = ''
  form.component = ''
  form.icon = ''
  form.sort = 0
  form.visible = 1
  form.status = 0
}

function openCreate(parent?: MenuRow): void {
  dialogMode.value = 'create'
  editingId.value = ''
  resetForm()
  if (parent) {
    form.parent_id = parent.id
    // UX 默认值：C 下新增默认按钮，其余默认菜单（无后端约束，仅省一次点选）
    form.menu_type = parent.menu_type === 'C' ? 'F' : 'C'
  }
  dialogVisible.value = true
}

function openEdit(row: MenuRow): void {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.parent_id = row.parent_id ?? ''
  form.menu_type = row.menu_type
  form.name = row.name
  form.perm_code = row.perm_code
  form.path = row.path
  form.component = row.component
  form.icon = row.icon
  form.sort = row.sort
  form.visible = row.visible
  form.status = row.status
  dialogVisible.value = true
}

function buildPayload(): MenuPayload {
  return {
    // create：空串=挂根；update：空串=移到根 / hashid=移动（同值后端判 no-op，不触发移动）
    parent_id: form.parent_id,
    menu_type: form.menu_type,
    // M/C 后端禁填 perm_code（ErrMenuPermForbidden），切类型后强制提交空串
    perm_code: form.menu_type === 'F' ? form.perm_code : '',
    name: form.name,
    path: form.path,
    component: form.component,
    icon: form.icon,
    sort: form.sort,
    visible: form.visible,
    status: form.status,
  }
}

async function submitForm(): Promise<void> {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  try {
    if (dialogMode.value === 'create') {
      await createMenu(buildPayload())
    } else {
      await updateMenu(editingId.value, buildPayload())
    }
  } catch {
    return // 请求层已 toast（如权限码已存在/无效父节点）；保留弹窗供修正
  }
  ElMessage.success(dialogMode.value === 'create' ? '新增成功' : '保存成功')
  dialogVisible.value = false
  fetchTree()
}

async function onDelete(row: MenuRow): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确定删除「${row.name}」？删除后该菜单及其角色授权关联将一并移除，不可恢复。` +
        '若存在子节点，后端将拒绝删除（需先删除子节点）。',
      '提示',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return // 用户取消
  }
  try {
    await removeMenu(row.id)
  } catch {
    return // 请求层已 toast（如"菜单下有子节点"）
  }
  ElMessage.success('删除成功')
  fetchTree()
}

onMounted(fetchTree)
</script>

<template>
  <el-card shadow="never">
    <div class="menu-toolbar">
      <el-button v-permission="'sys:menu:create'" type="primary" :icon="Plus" @click="openCreate()">
        新增菜单
      </el-button>
      <el-button :icon="Sort" @click="expandAll = !expandAll">
        {{ expandAll ? '折叠全部' : '展开全部' }}
      </el-button>
    </div>

    <el-table
      :key="tableKey"
      v-loading="loading"
      :data="tree"
      row-key="id"
      border
      :default-expand-all="expandAll"
      :tree-props="{ children: 'children' }"
    >
      <el-table-column prop="name" label="名称" min-width="200" show-overflow-tooltip />
      <el-table-column label="类型" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="typeMeta[row.menu_type]?.tag || 'info'" size="small">
            {{ typeMeta[row.menu_type]?.label || row.menu_type }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="sort" label="排序" width="70" align="center" />
      <el-table-column prop="perm_code" label="权限码" min-width="160" show-overflow-tooltip />
      <el-table-column prop="path" label="路由路径" min-width="120" show-overflow-tooltip />
      <el-table-column prop="component" label="组件" min-width="150" show-overflow-tooltip />
      <el-table-column prop="icon" label="图标" width="100" show-overflow-tooltip />
      <el-table-column label="可见" width="70" align="center">
        <template #default="{ row }">
          <template v-if="row.menu_type !== 'F'">{{ row.visible === 1 ? '显示' : '隐藏' }}</template>
          <template v-else>—</template>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="70" align="center">
        <template #default="{ row }">{{ row.status === 0 ? '正常' : '停用' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="210" fixed="right">
        <template #default="{ row }">
          <el-button
            v-permission="'sys:menu:create'"
            link
            type="primary"
            @click="openCreate(row)"
          >
            新增子级
          </el-button>
          <el-button v-permission="'sys:menu:update'" link type="primary" @click="openEdit(row)">
            编辑
          </el-button>
          <el-button v-permission="'sys:menu:delete'" link type="danger" @click="onDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <el-form-item label="父级菜单" prop="parent_id">
          <el-tree-select
            v-model="form.parent_id"
            :data="parentOptions"
            :props="{ label: 'name', children: 'children' }"
            node-key="id"
            value-key="id"
            check-strictly
            clearable
            placeholder="不选 = 根级"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="菜单类型" prop="menu_type">
          <el-radio-group v-model="form.menu_type">
            <el-radio-button value="M">目录</el-radio-button>
            <el-radio-button value="C">菜单</el-radio-button>
            <el-radio-button value="F">按钮</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="菜单显示名" />
        </el-form-item>
        <el-form-item v-if="showPermCode" label="权限码" prop="perm_code">
          <el-input v-model="form.perm_code" placeholder="如 sys:user:list（F 必填且唯一）" />
        </el-form-item>
        <el-form-item v-if="showPath" label="路由路径" prop="path">
          <el-input v-model="form.path" placeholder="如 /sys/user" />
        </el-form-item>
        <el-form-item v-if="showComponent" label="组件" prop="component">
          <el-input v-model="form.component" placeholder="如 sys/user/index（views 下相对路径）" />
        </el-form-item>
        <el-form-item v-if="showIcon" label="图标" prop="icon">
          <el-input v-model="form.icon" placeholder="图标名（如 setting/user/tree-table）" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="form.sort" controls-position="right" />
        </el-form-item>
        <el-form-item v-if="showVisible" label="可见" prop="visible">
          <el-radio-group v-model="form.visible">
            <el-radio :value="1">显示</el-radio>
            <el-radio :value="0">隐藏</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-radio-group v-model="form.status">
            <el-radio :value="0">正常</el-radio>
            <el-radio :value="1">停用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped lang="scss">
.menu-toolbar {
  margin-bottom: 12px;
  display: flex;
  gap: 8px;
  align-items: center;
}
</style>
