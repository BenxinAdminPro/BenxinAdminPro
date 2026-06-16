<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   部门管理 — parent_id 自引用树（el-table 内建 tree）+ 静态字段 CRUD + 移动节点
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-16 21:20:34
  | @updated   2026-06-16 21:48:30  T-012 增量：创建时间列 ISO→YYYY-MM-DD HH:mm:ss 格式化（镜像 post 页 dateText）
  +----------------------------------------------------------------------
  说明：后端 GET /sys/depts/tree 返回【已嵌套树】（服务端 buildTree，sort ASC, id ASC），
       表格直接消费后端树（零再组装、不引入顺序分歧）；utils/tree.ts 的 buildTree 用于
       父级选择器——编辑时 flatten → 排除自己及子孙(subtreeIds) → 重建（前端先挡"挂到自己
       子孙下"，后端 dept_service.go ancestors 校验仍是权威，前端不冒充安全）。
       buildTree 默认键零配（id / parent_id / children 与后端出参逐字对应）。
       部门为单一结构（无 menu_type 多态），表单仅 父级/名称/负责人/排序/状态 静态字段。
       update 为全量覆写 name/sort/leader/status（updateDeptReq.Name 无 binding:required）——
       编辑表单全字段回填 + 全量提交，漏 leader 即每次编辑静默清空部门负责人（命门，对标
       T-007h §8-3）。parent_id 三态：缺省=不移动 / 空串=移到根 / hashid=移动（同值后端 no-op）。
       后端树接口无任何查询参数——本页不伪造前端假搜索，诚实降级（同菜单页）。
       删除软删：有子部门→409 ErrDeptHasChildren、有归属用户→409 ErrDeptHasUsers，二者均拒删，
       确认措辞如实覆盖两种拒删情形（区别于菜单页仅"子节点"单条）。
-->
<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Sort } from '@element-plus/icons-vue'
import {
  getDeptTree,
  createDept,
  updateDept,
  removeDept,
  type DeptNode,
  type DeptPayload,
} from '@/api/dept'
import { buildTree, flattenTree, subtreeIds } from '@/utils/tree'

// 创建时间格式化：ISO（2026-06-08T16:16:33+08:00）→ 2026-06-08 16:16:33（镜像 post/index.vue:21 逐字相同）
const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')

// ---- 树数据（直接消费后端嵌套树）----
const loading = ref(false)
const tree = ref<DeptNode[]>([])

async function fetchTree(): Promise<void> {
  loading.value = true
  try {
    tree.value = (await getDeptTree()) || []
  } catch {
    // 请求层已按错误码 toast；保留旧数据
  } finally {
    loading.value = false
  }
}

// ---- 展开/折叠全部（经 :key 重建表格生效）----
const expandAll = ref(true)
const tableKey = computed(() => (expandAll.value ? 'expand' : 'collapse'))

// ---- 弹窗表单（部门无多态，固定字段）----
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref('')
const formRef = ref<FormInstance>()
const form = reactive({
  parent_id: '' as string,
  name: '',
  leader: '',
  sort: 0,
  status: 0,
})

const dialogTitle = computed(() => (dialogMode.value === 'create' ? '新增部门' : '编辑部门'))

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: '请输入部门名称', trigger: 'blur' }],
}))

// 父级选择器选项：编辑时排除"自己及子孙"（防挂到自己子树下；后端 ancestors 校验仍是权威）。
// buildTree 真消费点：flatten 后过滤重建（默认键 id/parent_id/children 零配）。
const parentOptions = computed<DeptNode[]>(() => {
  if (dialogMode.value !== 'edit' || !editingId.value) return tree.value
  const excluded = subtreeIds(tree.value, editingId.value)
  const flat = flattenTree(tree.value).filter((n) => !excluded.has(n.id))
  return buildTree(flat)
})

function resetForm(): void {
  form.parent_id = ''
  form.name = ''
  form.leader = ''
  form.sort = 0
  form.status = 0
}

function openCreate(parent?: DeptNode): void {
  dialogMode.value = 'create'
  editingId.value = ''
  resetForm()
  if (parent) {
    form.parent_id = parent.id // 行操作"新增子级"预填父级
  }
  dialogVisible.value = true
}

function openEdit(row: DeptNode): void {
  dialogMode.value = 'edit'
  editingId.value = row.id
  // 全字段回填（命门）：parent_id 三态 null→''（根）/ hashid→对应父，不移动时回传当前父=后端 no-op；
  // leader 必须回填，否则全量覆写时静默清空（T-007h §8-3）。
  form.parent_id = row.parent_id ?? ''
  form.name = row.name
  form.leader = row.leader
  form.sort = row.sort
  form.status = row.status
  dialogVisible.value = true
}

function buildPayload(): DeptPayload {
  return {
    // create：空串=挂根；update：空串=移到根 / hashid=移动（同值后端判 no-op，不触发移动）
    parent_id: form.parent_id,
    name: form.name,
    leader: form.leader,
    sort: form.sort,
    status: form.status,
  }
}

async function submitForm(): Promise<void> {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  try {
    if (dialogMode.value === 'create') {
      await createDept(buildPayload())
    } else {
      await updateDept(editingId.value, buildPayload())
    }
  } catch {
    return // 请求层已 toast（如无效父节点/防环 400）；保留弹窗供修正
  }
  ElMessage.success(dialogMode.value === 'create' ? '新增成功' : '保存成功')
  dialogVisible.value = false
  fetchTree()
}

async function onDelete(row: DeptNode): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确定删除「${row.name}」？删除后该部门将被移除，不可恢复。` +
        '若该部门下有子部门或有归属用户，后端将拒绝删除（需先处理子部门 / 转移用户）。',
      '提示',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return // 用户取消
  }
  try {
    await removeDept(row.id)
  } catch {
    return // 请求层已 toast（如"部门下有子部门" / "部门下有用户"）
  }
  ElMessage.success('删除成功')
  fetchTree()
}

onMounted(fetchTree)
</script>

<template>
  <el-card shadow="never">
    <div class="dept-toolbar">
      <el-button v-permission="'sys:dept:create'" type="primary" :icon="Plus" @click="openCreate()">
        新增部门
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
      <el-table-column prop="name" label="部门名称" min-width="220" show-overflow-tooltip />
      <el-table-column prop="leader" label="负责人" min-width="120" show-overflow-tooltip />
      <el-table-column prop="sort" label="排序" width="80" align="center" />
      <el-table-column label="状态" width="80" align="center">
        <template #default="{ row }">{{ row.status === 0 ? '正常' : '停用' }}</template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="170" show-overflow-tooltip>
        <template #default="{ row }">{{ dateText(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="210" fixed="right">
        <template #default="{ row }">
          <el-button
            v-permission="'sys:dept:create'"
            link
            type="primary"
            @click="openCreate(row)"
          >
            新增子级
          </el-button>
          <el-button v-permission="'sys:dept:update'" link type="primary" @click="openEdit(row)">
            编辑
          </el-button>
          <el-button v-permission="'sys:dept:delete'" link type="danger" @click="onDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <el-form-item label="父级部门" prop="parent_id">
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
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="部门显示名" />
        </el-form-item>
        <el-form-item label="负责人" prop="leader">
          <el-input v-model="form.leader" placeholder="部门负责人（可空）" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="form.sort" controls-position="right" />
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
.dept-toolbar {
  margin-bottom: 12px;
  display: flex;
  gap: 8px;
  align-items: center;
}
</style>
