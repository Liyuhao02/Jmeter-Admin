import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layout/MainLayout.vue'
import ScriptList from '@/views/ScriptList.vue'
import ScriptEdit from '@/views/ScriptEdit.vue'
import SlaveManage from '@/views/SlaveManage.vue'
import ExecutionList from '@/views/ExecutionList.vue'
import ExecutionDetail from '@/views/ExecutionDetail.vue'

const routes = [
  {
    path: '/',
    component: MainLayout,
    redirect: '/scripts',
    children: [
      {
        path: 'scripts',
        name: 'ScriptList',
        component: ScriptList,
        meta: { title: '脚本管理' }
      },
      {
        path: 'scripts/:id/edit',
        name: 'ScriptEdit',
        component: ScriptEdit,
        meta: { title: '编辑脚本' }
      },
      {
        path: 'slaves',
        name: 'SlaveManage',
        component: SlaveManage,
        meta: { title: 'Slave管理' }
      },
      {
        path: 'executions',
        name: 'ExecutionList',
        component: ExecutionList,
        meta: { title: '执行记录' }
      },
      {
        path: 'executions/:id',
        name: 'ExecutionDetail',
        component: ExecutionDetail,
        meta: { title: '执行详情' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
