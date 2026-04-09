<template>
  <router-view v-slot="{ Component }">
    <transition name="app-fade" mode="out-in">
      <component :is="Component" />
    </transition>
  </router-view>
</template>

<style>
/* 全局应用过渡 - 更柔和的效果 */
.app-fade-enter-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.app-fade-leave-active {
  transition: opacity 0.2s ease;
}

.app-fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.app-fade-leave-to {
  opacity: 0;
}
</style>
