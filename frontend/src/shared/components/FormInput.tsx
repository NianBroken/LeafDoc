// 为受控文本输入统一提供可选标题、错误样式和错误文字，不在内部保存输入值。
import { Input } from '@/shared/components/ui/input'

// allowedHotkeys 透传为 data-allow-hotkeys，实际键盘放行由全局快捷键规则读取。
type FormInputProps = {
  label?: string
  name?: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  testid?: string
  error?: string
  type?: string
  autoComplete?: string
  autoFocus?: boolean
  readOnly?: boolean
  allowedHotkeys?: string
}

// FormInput 把事件中的字符串值传给调用方，不执行裁空白、格式转换或提交校验。
// label 仅渲染文字标签，本组件未生成 id 与 htmlFor 的显式关联。
export function FormInput({
  label,
  name,
  value,
  autoComplete,
  autoFocus,
  onChange,
  testid,
  placeholder,
  error,
  type = 'text',
  readOnly = false,
  allowedHotkeys,
}: FormInputProps) {
  return (
    <div className="form-input">
      {label && <label className="form-input__label">{label}</label>}
      <Input
        autoFocus={autoFocus || false}
        name={name}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete={autoComplete}
        readOnly={readOnly}
        className={error ? 'form-input__input-error' : ''}
        data-testid={testid}
        data-allow-hotkeys={allowedHotkeys}
      />
      {error && <p className="form-input__error">{error}</p>}
    </div>
  )
}
