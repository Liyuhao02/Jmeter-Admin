import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layout/MainLayout.vue'

const ScriptList = () => import('@/views/ScriptList.vue')
const ScriptEdit = () => import('@/views/ScriptEdit.vue')
const ScriptExecute = () => import('@/views/ScriptExecute.vue')
const SlaveManage = () => import('@/views/SlaveManage.vue')
const ExecutionList = () => import('@/views/ExecutionList.vue')
const ExecutionDetail = () => import('@/views/ExecutionDetail.vue')

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
        path: 'scripts/:id/execute',
        name: 'ScriptExecute',
        component: ScriptExecute,
        meta: { title: '执行脚本' }
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
