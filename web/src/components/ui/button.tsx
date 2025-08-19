import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const buttonVariants = cva(
	'btn px-4 py-2 bg-primary text-primary-foreground hover:opacity-90',
	{
		variants: {
			variant: {
				default: '',
				outline:
					'bg-transparent border border-primary text-primary hover:bg-primary hover:text-primary-foreground',
				ghost: 'bg-transparent hover:bg-card',
			},
			size: {
				default: 'h-10',
				sm: 'h-9 px-3',
				lg: 'h-11 px-8',
			},
		},
		defaultVariants: {
			variant: 'default',
			size: 'default',
		},
	},
)

export interface ButtonProps
	extends React.ButtonHTMLAttributes<HTMLButtonElement>,
		VariantProps<typeof buttonVariants> {}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
	({ className, variant, size, ...props }, ref) => {
		return (
			<button className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />
		)
	},
)
Button.displayName = 'Button'

export { Button, buttonVariants }


