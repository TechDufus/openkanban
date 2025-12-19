import "solid-js"

declare module "solid-js" {
  namespace JSX {
    interface IntrinsicElements {
      box: BoxProps
      text: TextProps
    }

    interface BoxProps {
      flexDirection?: "row" | "column"
      flex?: number
      width?: string | number
      height?: string | number
      padding?: number
      paddingTop?: number
      paddingBottom?: number
      paddingLeft?: number
      paddingRight?: number
      margin?: number
      marginTop?: number
      marginBottom?: number
      marginLeft?: number
      marginRight?: number
      gap?: number
      justifyContent?: "flex-start" | "center" | "flex-end" | "space-between" | "space-around"
      alignItems?: "flex-start" | "center" | "flex-end" | "stretch"
      backgroundColor?: string
      borderStyle?: "single" | "double" | "rounded" | "bold" | "none"
      borderColor?: string
      overflow?: "visible" | "hidden" | "scroll"
      children?: JSX.Element
    }

    interface TextProps {
      color?: string
      backgroundColor?: string
      bold?: boolean
      italic?: boolean
      underline?: boolean
      strikethrough?: boolean
      wrap?: "wrap" | "truncate" | "truncate-end" | "truncate-middle" | "truncate-start"
      children?: JSX.Element | string | number | (string | number)[]
    }
  }
}
