package gold

type RenderContext struct {
	style      map[StyleType]ElementStyle
	blockStack *BlockStack
	table      *TableElement
}
