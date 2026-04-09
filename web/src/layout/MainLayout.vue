<template>
  <div class="main-layout">
    <!-- 顶部导航栏 -->
    <header class="top-header">
      <div class="header-content">
        <!-- 左侧 Logo -->
        <div class="logo">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" fill="url(#logoGradient)" />
              <defs>
                <linearGradient id="logoGradient" x1="3" y1="2" x2="21" y2="22" gradientUnits="userSpaceOnUse">
                  <stop stop-color="#0066ff" />
                  <stop offset="1" stop-color="#00aaff" />
                </linearGradient>
              </defs>
            </svg>
          </div>
          <span class="logo-text">JMeter Admin</span>
        </div>

        <!-- 右侧 Tab 导航 -->
        <nav class="tab-nav">
          <router-link
            v-for="tab in tabs"
            :key="tab.path"
            :to="tab.path"
            class="tab-item"
            :class="{ active: isActive(tab.path) }"
          >
            <component :is="tab.icon" class="tab-icon" />
            <span>{{ tab.name }}</span>
          </router-link>
        </nav>
      </div>
    </header>

    <!-- 主内容区域 -->
    <main class="main-content">
      <div class="content-wrapper" :class="{ 'content-wrapper--wide': isScriptEditPage }">
        <router-view v-slot="{ Component }">
          <transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>

    <!-- 底部版本信息 -->
    <footer class="footer">
      <span>JMeter Admin v1.0.0</span>
    </footer>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Document, Monitor, DataLine } from '@element-plus/icons-vue'

const route = useRoute()

const tabs = [
  { name: '脚本管理', path: '/scripts', icon: Document },
  { name: 'Slave管理', path: '/slaves', icon: Monitor },
  { name: '执行记录', path: '/executions', icon: DataLine }
]

const isActive = (path) => {
  return route.path.startsWith(path)
}

const isScriptEditPage = computed(() => {
  return /^\/scripts\/[^/]+\/edit$/.test(route.path)
})
</script>

<style scoped lang="scss">
.main-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background-color: #0a0e17;
}

// 顶部导航栏
.top-header {
  height: 64px;
  background: #111827;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  position: sticky;
  top: 0;
  z-index: 100;
  flex-shrink: 0;
}

.header-content {
  max-width: 1400px;
  margin: 0 auto;
  height: 100%;
  padding: 0 32px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

// Logo 样式
.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  width: 28px;
  height: 28px;

  svg {
    width: 100%;
    height: 100%;
  }
}

.logo-text {
  font-size: 17px;
  font-weight: 600;
  color: #ffffff;
  letter-spacing: 0.5px;
}

// Tab 导航
.tab-nav {
  display: flex;
  align-items: center;
  gap: 4px;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.6);
  text-decoration: none;
  transition: all 0.25s ease;
  position: relative;

  &:hover {
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.85);
  }

  &.active {
    background: rgba(0, 170, 255, 0.1);
    color: #ffffff;

    &::after {
      content: '';
      position: absolute;
      bottom: -1px;
      left: 50%;
      transform: translateX(-50%);
      width: 24px;
      height: 2px;
      background: #00aaff;
      border-radius: 1px;
    }
  }

  .tab-icon {
    width: 16px;
    height: 16px;
  }
}

// 主内容区域
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  background-color: #0a0e17;
}

.content-wrapper {
  flex: 1;
  max-width: 1400px;
  width: 100%;
  margin: 0 auto;
  padding: 32px;
}

.content-wrapper--wide {
  max-width: none;
  padding: 16px 20px 20px;
}

// 底部
.footer {
  padding: 16px 32px;
  text-align: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.04);
  flex-shrink: 0;
}

// 页面过渡动画 - 柔和的淡入效果
.page-fade-enter-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.page-fade-leave-active {
  transition: opacity 0.2s ease;
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.page-fade-leave-to {
  opacity: 0;
}
</style>
