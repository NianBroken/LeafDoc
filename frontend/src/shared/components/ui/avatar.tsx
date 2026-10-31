// 头像根、图片与回退区域共用圆形布局，加载状态由 Radix 原语协调。
import * as React from 'react'
import * as AvatarPrimitive from '@radix-ui/react-avatar'

import { withBasePath } from '@/shared/lib/routePath'

import { cn } from '@/shared/lib/utils'

import { DEFAULT_AVATAR_PATH } from '@/shared/lib/brandDefaults'

// 根区域默认四十像素宽高并裁剪溢出，调用方可用合并类名覆盖尺寸。
const Avatar = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Root>
>(({ className, ...props }, ref) => (
  <AvatarPrimitive.Root
    ref={ref}
    className={cn(
      'relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full',
      className,
    )}
    {...props}
  />
))
Avatar.displayName = AvatarPrimitive.Root.displayName

// 图片来源、替代文字和加载属性透传，不在包装层再次请求或缓存图片。
const AvatarImage = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Image>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Image>
>(({ className, ...props }, ref) => (
  <AvatarPrimitive.Image
    ref={ref}
    className={cn('preview-image', className)}
    {...props}
  />
))
AvatarImage.displayName = AvatarPrimitive.Image.displayName

// 回退内容固定为带部署前缀的内置头像，不渲染调用方传入的 children。
// 内置图片的空 alt 表示装饰内容，账号名称等可访问文字由外围界面提供。
const AvatarFallback = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Fallback>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Fallback>
>(({ className, children: _children, ...props }, ref) => (
  <AvatarPrimitive.Fallback
    ref={ref}
    className={cn(
      'bg-muted-surface flex h-full w-full items-center justify-center rounded-full',
      className,
    )}
    {...props}
  >
    <img
      src={withBasePath(DEFAULT_AVATAR_PATH)}
      alt=""
      className="preview-image"
    />
  </AvatarPrimitive.Fallback>
))
AvatarFallback.displayName = AvatarPrimitive.Fallback.displayName

export { Avatar, AvatarImage, AvatarFallback }
