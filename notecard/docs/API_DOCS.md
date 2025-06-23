

FUNCTIONS

func CreateAPIResources() []mcp.Resource
    CreateAPIResources creates multiple Notecard API documentation resources,
    one for each category

func CreateDecryptNoteTool() mcp.Tool
func CreateNotecardGetAPIsTool() mcp.Tool
func CreateNotecardInitializeTool() mcp.Tool
func CreateNotecardListFirmwareVersionsTool() mcp.Tool
func CreateNotecardRequestTool() mcp.Tool
func CreateNotecardUpdateFirmwareTool() mcp.Tool
func CreateNotecardValidateRequestTool() mcp.Tool
func CreateProvisionNotecardTool() mcp.Tool
func CreateSendNoteTool() mcp.Tool
func CreateTroubleshootConnectionTool() mcp.Tool
func HandleAPIResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error)
    HandleAPIResource handles requests for category-specific API documentation
    resources

