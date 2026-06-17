<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   角色管理 — x-table 配置化 CRUD + 分配菜单授权树（el-tree check-strictly 全量覆写）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  | @updated   2026-06-14 18:30:00  T-008c：+分配菜单行操作（el-tree 授权树回填全量+check-strictly+全量覆写提交）
  +----------------------------------------------------------------------
  授权树口径（以后端源码为准，T-008c 摸底已坐实）：
   - 回填来源 GET /sys/roles/:id 出参 menu_ids = 当前【全量】已授（含 M/C/F 三层），hashid 全程透传不解码。
   - 提交 PUT /sys/roles/:id/menus 为【全量覆写】（service 先删后建 + 联动 Casbin p 规则）：
     回填必须返全集，漏勾的已授节点会在覆写时被静默删除 → 防误清是正确性前提。
   - check-strictly（父子各自独立勾选）：role_menu 扁平逐字存储 + 全量覆写，故采「回填集≡勾选集≡提交集」
     恒等往返（setCheckedKeys(menu_ids) ↔ getCheckedKeys()），从根上杜绝静默丢权限/越权扩权
     （级联模式半选父丢/setCheckedKeys 级联污染未授子的双坑，§摸底）。半选概念不存在，提交=getCheckedKeys()。
   - 已知 UX（非 bug）：勾页面 C 不自动勾父目录 M；只勾 C 不勾 M 时该页在侧边栏会平铺到顶层
     （GetUserMenuTree 仅渲染 role_menu 里的 M/C，孤儿 C 落根）。按目录整片勾即正常嵌套。
   - 防误清（吃 T-007h api.get 失败教训）：开弹窗先 Promise.all([getRole 取 menu_ids, getMenuTree 载树])，
     任一失败则不开弹窗、不提交，杜绝「空树/残缺回填 → 覆写 → 清光授权」。
-->
<script setup lang="ts">
import { ref, reactive, nextTick } from 'vue'
import { ElMessage, ElTree } from 'element-plus'
import { Operation } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'
import { listRoles, createRole, updateRole, removeRole, getRole, assignRoleMenus } from '@/api/role'
import { getMenuTree, type MenuRow } from '@/api/menu'
import { statusText } from '@/utils/format'

const scopeMap: Record<number, string> = { 1: '全部', 2: '本人', 3: '本部门' }
const scopeText = (v: unknown) => scopeMap[Number(v)] || '本人'

// ---- 分配菜单授权树弹窗（页级控件；check-strictly 独立勾选 + 全量覆写）----
const menuVisible = ref(false)
const menuTarget = reactive<{ id: string; name: string }>({ id: '', name: '' })
const menuTree = ref<MenuRow[]>([])
const menuSubmitting = ref(false)
const treeRef = ref<InstanceType<typeof ElTree>>()
const treeProps = { label: 'name', children: 'children' } as const

async function openAssignMenus(row: XRow): Promise<void> {
  const id = String(row.id)
  let detailMenuIds: string[] = []
  try {
    // 回填以「当前已授全量」为基准 + 并行载全量菜单树；任一失败不开弹窗（防误清）。
    const [detail, tree] = await Promise.all([getRole(id), getMenuTree()])
    detailMenuIds = detail.menu_ids ?? []
    menuTree.value = tree
  } catch {
    return
  }
  menuTarget.id = id
  menuTarget.name = String(row.name ?? '')
  menuVisible.value = true
  // 树挂载后按全量已授勾回（check-strictly：无级联，恒等回填）。
  await nextTick()
  treeRef.value?.setCheckedKeys(detailMenuIds, false)
}

async function submitAssignMenus(): Promise<void> {
  // check-strictly 下半选恒空 → getCheckedKeys() 即全集（≡回填集语义），直接全量覆写提交。
  const keys = (treeRef.value?.getCheckedKeys(false) ?? []) as string[]
  menuSubmitting.value = true
  try {
    await assignRoleMenus(menuTarget.id, keys)
    ElMessage.success('授权已保存')
    menuVisible.value = false
  } catch {
    // 请求层已 toast；保留弹窗供修正，不冒未处理 rejection
  } finally {
    menuSubmitting.value = false
  }
}

const config: XTableConfig = {
  permPrefix: 'sys:role',
  actionsWidth: 240, // 容纳 编辑/删除/分配菜单
  actions: [
    {
      label: '分配菜单',
      perm: 'sys:role:assign',
      type: 'primary',
      icon: Operation,
      handler: (row: XRow) => openAssignMenus(row),
    },
  ],
  api: {
    list: listRoles,
    create: createRole,
    update: updateRole,
    remove: removeRole,
  },
  columns: [
    { prop: 'code', label: '角色编码', minWidth: 140 },
    { prop: 'name', label: '角色名称', minWidth: 120 },
    { prop: 'data_scope', label: '数据范围', width: 100, formatter: (_r, v) => scopeText(v) },
    { prop: 'sort', label: '排序', width: 80 },
    { prop: 'status', label: '状态', width: 90, formatter: (_r, v) => statusText(v) },
  ],
  fields: [
    { prop: 'code', label: '角色编码', required: true, editable: false },
    { prop: 'name', label: '角色名称', required: true },
    {
      prop: 'data_scope',
      label: '数据范围',
      type: 'select',
      default: 2,
      options: [
        { label: '全部', value: 1 },
        { label: '本人', value: 2 },
        { label: '本部门', value: 3 },
      ],
    },
    { prop: 'sort', label: '排序', type: 'number', default: 0 },
    {
      prop: 'status',
      label: '状态',
      type: 'select',
      default: 0,
      options: [
        { label: '正常', value: 0 },
        { label: '停用', value: 1 },
      ],
    },
    { prop: 'remark', label: '备注', type: 'textarea' },
  ],
}
</script>

<template>
  <el-card shadow="never">
    <XTable :config="config" />

    <!-- 分配菜单授权树（check-strictly 独立勾选；回填全量已授 → 改 → 全量覆写 PUT :id/menus） -->
    <el-dialog
      v-model="menuVisible"
      :title="`分配菜单 - ${menuTarget.name}`"
      width="460px"
      destroy-on-close
    >
      <el-tree
        ref="treeRef"
        :data="menuTree"
        :props="treeProps"
        node-key="id"
        show-checkbox
        check-strictly
        default-expand-all
        class="menu-auth-tree"
      />
      <div class="menu-assign-hint">
        独立勾选（父子不联动）；提交后以当前勾选全量覆写该角色授权（含 Casbin 权限联动）。
      </div>
      <template #footer>
        <el-button @click="menuVisible = false">取消</el-button>
        <el-button type="primary" :loading="menuSubmitting" @click="submitAssignMenus">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped>
.menu-auth-tree {
  max-height: 50vh;
  overflow-y: auto;
}
.menu-assign-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
