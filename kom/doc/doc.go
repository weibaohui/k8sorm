package doc

import (
	"encoding/json"
	"strings"

	"github.com/google/gnostic-models/openapiv2"
	"github.com/weibaohui/kom/utils"
	"k8s.io/klog/v2"
)

var trees []TreeNode

type Docs struct {
	Trees []TreeNode
}

// TreeNode 表示树形结构的节点
type TreeNode struct {
	ID              string      `json:"id"` // GVK形式io.k8s.apimachinery.pkg.apis.meta.v1.MicroTime
	Label           string      `json:"label"`
	Value           string      `json:"value"` // amis tree 需要
	Description     string      `json:"description,omitempty"`
	Type            string      `json:"type,omitempty"`
	Ref             string      `json:"ref,omitempty"`
	Enum            []Enum      `json:"enum,omitempty"`
	Items           Items       `json:"items,omitempty"`
	VendorExtension interface{} `json:"vendor_extension,omitempty"`
	Children        []*TreeNode `json:"children,omitempty"`
	group           string      // 从ID中尝试解析GVK，方便查询，不一定准确
	version         string      // 从ID中尝试解析GVK，方便查询
	kind            string      // 从ID中尝试解析GVK，方便查询
}

// SchemaDefinition 表示根定义
type SchemaDefinition struct {
	Name  string      `json:"name"`
	Value SchemaValue `json:"value"`
}

// SchemaValue 表示定义的值
type SchemaValue struct {
	Description     string           `json:"description"`
	Properties      SchemaProperties `json:"properties"`
	Type            SchemaType       `json:"type"`
	VendorExtension []interface{}    `json:"vendor_extension,omitempty"`
}

// SchemaProperties 表示属性
type SchemaProperties struct {
	AdditionalProperties []Property `json:"additional_properties"`
}

// Property 表示单个属性
type Property struct {
	Name  string        `json:"name"`
	Value PropertyValue `json:"value"`
}

// PropertyValue 表示属性的值
type PropertyValue struct {
	Description     string           `json:"description,omitempty"`
	Type            *SchemaType      `json:"type,omitempty"`
	Ref             string           `json:"_ref,omitempty"`
	Enum            []Enum           `json:"enum,omitempty"`
	Items           Items            `json:"items,omitempty"`
	VendorExtension interface{}      `json:"vendor_extension,omitempty"`
	Properties      SchemaProperties `json:"properties"`
}
type Enum struct {
	Yaml string `json:"yaml,omitempty"`
}
type Schema struct {
	Ref string `json:"_ref,omitempty"`
}
type Items struct {
	Schema []Schema `json:"schema,omitempty"`
}

// SchemaType 表示类型
type SchemaType struct {
	Value []string `json:"value,omitempty"`
}

// RootDefinitions 最外层定义
type RootDefinitions struct {
	Swagger     string      `json:"swagger"`
	Definitions Definitions `json:"definitions,omitempty"`
}

// Definitions 表示所有定义
// 使用interface{}
type Definitions struct {
	AdditionalProperties []map[string]interface{} `json:"additional_properties"`
}

// definitionsMap 存储所有定义，以便处理引用
var definitionsMap map[string]SchemaDefinition

// parseOpenAPISchema 解析 OpenAPI Schema JSON 字符串并返回根 TreeNode
// Example:
//
//	  JSON样例
//		 "name": "com.example.stable.v1.CronTab",
//			"value": { },
//			"properties": {
//			    "additional_properties": [ {},{}]
//			  },
//			  "vendor_extension": [ {},{}]
//			}
func parseOpenAPISchema(schemaJSON string, flag map[string]struct{}) (TreeNode, error) {
	var def SchemaDefinition
	err := json.Unmarshal([]byte(schemaJSON), &def)
	if err != nil {
		return TreeNode{}, err
	}
	// klog.V(2).Infof("add def cache %s", def.Name)
	definitionsMap[def.Name] = def
	// klog.V(2).Infof("add def length %d", len(definitionsMap))

	return buildTree(def, flag), nil
}
func parseID(id string) (group, version, kind string) {
	parts := strings.Split(id, ".")
	if len(parts) < 3 {
		return "", "", "" // 不足三个部分，无法解析
	}

	kind = parts[len(parts)-1]    // 最后一段是 Kind
	version = parts[len(parts)-2] // 倒数第二段是 Version

	if len(parts) > 3 { // 判断是否有 Group 部分
		groupParts := parts[:len(parts)-2] // Group 是前面的部分
		group = groupParts[len(groupParts)-1]
	}
	if group == "core" {
		// core 在书写yaml时不需要写，但是解析出来还是有core，这里做一下处理
		group = ""
	}

	return group, version, kind
}

// buildTree 根据 SchemaDefinition 构建 TreeNode
func buildTree(def SchemaDefinition, flag map[string]struct{}) TreeNode {
	// todo 应该使用GVK作为
	labelParts := strings.Split(def.Name, ".")
	label := labelParts[len(labelParts)-1]

	nodeType := ""
	if len(def.Value.Type.Value) > 0 {
		nodeType = def.Value.Type.Value[0]
	}

	var children []*TreeNode
	for _, prop := range def.Value.Properties.AdditionalProperties {
		children = append(children, buildPropertyNode(prop, flag))
	}

	group, version, kind := parseID(def.Name)
	return TreeNode{
		ID:          def.Name,
		Label:       label,
		Value:       utils.RandNLengthString(20),
		Description: def.Value.Description,
		Type:        nodeType,
		Children:    children,
		group:       group,
		version:     version,
		kind:        kind,
	}
}

// buildPropertyNode 根据 Property 构建 TreeNode
func buildPropertyNode(prop Property, flag map[string]struct{}) *TreeNode {
	label := prop.Name
	nodeID := prop.Name
	description := prop.Value.Description
	nodeType := ""
	ref := ""

	if prop.Value.Type != nil && len(prop.Value.Type.Value) > 0 {
		nodeType = prop.Value.Type.Value[0]
	}
	if prop.Value.Ref != "" {
		ref = prop.Value.Ref
	}

	var children []*TreeNode

	// 如果有引用，查找定义并递归构建子节点
	if ref != "" {
		if _, ok := flag[ref]; ok {
			klog.V(8).Infof("buildPropertyNode for ref: %s already processed", ref)
		} else {
			klog.V(8).Infof("buildPropertyNode for ref: %s", ref)

			flag[ref] = struct{}{} // 标记为已处理
			// 假设 ref 的格式为 "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
			refParts := strings.Split(ref, "/")
			refName := refParts[len(refParts)-1]
			// 构建完整的引用路径
			// fullRef := strings.Join(refParts[1:], ".")

			// 这个可能会导致 循环引用溢出
			if def, exists := definitionsMap[refName]; exists {
				childNode := buildTree(def, flag)
				children = append(children, &childNode)
			} else {
				// 如果引用的定义不存在，可以记录为一个叶子节点或处理为需要进一步扩展
				children = append(children, &TreeNode{
					ID:          refName,
					Label:       refName,
					Value:       refName,
					Description: "Referenced definition not found",
				})
			}
		}

	}

	for _, pp := range prop.Value.Properties.AdditionalProperties {
		children = append(children, buildPropertyNode(pp, flag))
	}

	return &TreeNode{
		ID:          nodeID,
		Label:       label,
		Value:       nodeID,
		Description: description,
		Type:        nodeType,
		Ref:         ref,
		Items:       prop.Value.Items,
		Enum:        prop.Value.Enum,
		Children:    children,
	}
}

// printTree 递归打印 TreeNode
func printTree(node *TreeNode, level int) {
	indent := strings.Repeat("  ", level)
	klog.V(2).Infof("%s%s (ID: %s)\n", indent, node.Label, node.ID)
	if node.Description != "" {
		klog.V(2).Infof("%s  Description: %s\n", indent, node.Description)
	}
	if node.Type != "" {
		klog.V(2).Infof("%s  Type: %s\n", indent, node.Type)
	}
	if node.Ref != "" {
		klog.V(2).Infof("%s  Ref: %s\n", indent, node.Ref)
	}

	for _, child := range node.Children {
		printTree(child, level+1)
	}
}

func InitTrees(schema *openapi_v2.Document) *Docs {
	definitionsMap = make(map[string]SchemaDefinition)
	refFlag := make(map[string]struct{})    // 记录ref是否已经被处理，存在key 即为已处理，不存在即为未处理
	refArrayNodes := map[string]*TreeNode{} // loadArrayItems 时记录已经处理过的ref节点

	// 将 OpenAPI Schema 转换为 JSON 字符串
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		klog.V(2).Infof("Error marshaling OpenAPI schema to JSON: %v\n", err)
		return nil
	}
	// os.WriteFile("def.json", schemaBytes, 0644)
	// 打印部分 Schema 以供调试
	// klog.V(2).Infof(string(schemaBytes))

	root := &RootDefinitions{}
	err = json.Unmarshal(schemaBytes, root)
	if err != nil {
		klog.V(2).Infof("Error unmarshaling OpenAPI schema: %v\n", err)
		return nil
	}
	definitionList := root.Definitions.AdditionalProperties

	// 进行第一遍处理，此时Ref并没有读取，只是记录了引用
	for _, definition := range definitionList {
		str := utils.ToJSON(definition)
		// 解析 Schema 并构建树形结构
		treeRoot, err := parseOpenAPISchema(str, refFlag)
		if err != nil {
			klog.V(2).Infof("Error parsing OpenAPI schema: %v\n", err)
			continue
		}
		trees = append(trees, treeRoot)
	}

	// 进行遍历处理，将child中ref对应的类型提取出来
	// 此时应该所有的类型都已经存在了
	for _, item := range trees {
		loadChild(&item, refArrayNodes)
	}

	for _, item := range trees {
		loadArrayItems(&item, refArrayNodes)
	}

	// 此时 层级结构当中是ref 下面是具体的一个结构体A
	// 结构体A的child是各个属性
	// 我们需要把child下的属性上提一级，避免出现A、再展开才是具体属性的情况
	for _, item := range trees {
		childMoveUpLevel(&item)
	}

	// 处理Array Items的情况
	// "items": {
	//   "schema": [
	//     {
	//        "_ref": "#/definitions/io.k8s.api.core.v1.Container"
	//     }
	//    ]
	// }

	// 将所有节点的ID，改为唯一的
	for _, item := range trees {
		uniqueID(&item)
	}

	return &Docs{
		Trees: trees,
	}
}
func loadArrayItems(node *TreeNode, refArrayNodes map[string]*TreeNode) {

	if len(node.Items.Schema) > 0 && node.Items.Schema[0].Ref != "" {
		ref := node.Items.Schema[0].Ref
		if n, ok := refArrayNodes[ref]; ok {
			// 已经存在了
			node.Children = n.Children
		} else {
			// 第一次处理
			klog.V(6).Infof("loadArrayItems ref=%s", ref)
			refNode := FetchByRef(ref, refArrayNodes)
			refArrayNodes[ref] = refNode
			node.Children = refNode.Children
		}

	}
	for i := range node.Children {
		// 检测循环引用
		// 在构建树时，验证子节点是否已经存在于父节点的链路中
		treeNode := node.Children[i]
		if len(treeNode.Items.Schema) > 0 && treeNode.Items.Schema[0].Ref != "" {
			ref := treeNode.Items.Schema[0].Ref
			if _, ok := refArrayNodes[ref]; !ok {
				loadArrayItems(treeNode, refArrayNodes)
			}
		}

	}
}
func childMoveUpLevel(item *TreeNode) {
	name := strings.TrimPrefix(item.Ref, "#/definitions/")
	if item.Ref != "" && len(item.Children) == 1 && item.Children[0].ID == name && len(item.Children[0].Children) > 0 {

		item.Children = item.Children[0].Children
	}
	for i := range item.Children {
		childMoveUpLevel(item.Children[i])
	}
}
func loadChild(item *TreeNode, refArrayNodes map[string]*TreeNode) {
	name := strings.TrimPrefix(item.Ref, "#/definitions/")

	if item.Ref != "" && len(item.Children) > 0 && item.Children[0].ID == name {
		refNode := FetchByRef(item.Ref, refArrayNodes)
		item.Children[0] = refNode
	}
	for i := range item.Children {
		loadChild(item.Children[i], refArrayNodes)
	}
}
func uniqueID(item *TreeNode) {
	item.Value = utils.RandNLengthString(20)
	for i := range item.Children {
		uniqueID(item.Children[i])
	}
}

func (d *Docs) ListNames() {
	for _, tree := range d.Trees {
		klog.Infof("tree info ID: %s\tLabel:%s\t\n Parse GVK=[%s,%s,%s]", tree.ID, tree.Label, tree.group, tree.version, tree.kind)
	}
}
func FetchByRef(ref string, refArrayNodes map[string]*TreeNode) *TreeNode {
	// #/definitions/io.k8s.api.core.v1.PodSpec
	if n, ok := refArrayNodes[ref]; ok {
		// 已经存在了
		return n
	} else {
		id := strings.TrimPrefix(ref, "#/definitions/")
		for _, tree := range trees {
			if tree.ID == id {
				// 为了避免多个node引用同一个节点，需要深拷贝
				// 否则会有相同的value，前端显示会有点显示错位
				klog.V(6).Infof("FetchByRef %s\n", id)
				dcp, _ := utils.DeepCopy(tree)
				refArrayNodes[ref] = &dcp
				return &dcp
			}
		}
	}
	return nil
}
func (d *Docs) Fetch(kind string) *TreeNode {
	for _, tree := range d.Trees {
		if tree.Label == kind {
			return &tree
		}
	}
	return nil
}

// FetchByGVK
// com.example.stable.v1.CronTabList
// apiVersion: stable.example.com/v1
// kind: CronTab
func (d *Docs) FetchByGVK(apiVersion, kind string) (node *TreeNode) {
	// 先从 apiVersion+kind 查找，如果找不到再从 kind 查找
	// 采用HasSuffix来匹配,因为内置资源的apiVersion会省略前面的io.k8s.api.core等类似的前缀
	// "id": "io.k8s.api.core.v1.Namespace",
	// group：events.k8s.io =>io.k8s.api.events.v1.Event
	// group""=>io.k8s.api.core.v1.Event
	var group string
	var version string
	if !strings.Contains(apiVersion, "/") {
		group = ""
		version = apiVersion
	} else {
		parts := strings.Split(apiVersion, "/")
		if len(parts) == 2 {
			group = parts[0]
			version = parts[1]
			if strings.Contains(group, ".") {
				ps := strings.Split(group, ".")
				group = ps[0]
			}

		}
	}

	for _, tree := range d.Trees {
		if tree.version == version && tree.kind == kind && tree.group == group {
			node = &tree
			klog.V(6).Infof("[%s:%s]=>[%s,%s,%s]find node ID:%s \tBy GVK(%s,%s,%s)", apiVersion, kind, group, version, kind, tree.ID, tree.group, tree.version, tree.kind)
			return
		}
	}
	for _, tree := range d.Trees {
		if tree.version == version && tree.kind == kind {
			node = &tree
			klog.V(6).Infof("[%s:%s]=>[%s,%s,%s]find node ID:%s \tBy KV(%s,%s)", apiVersion, kind, group, version, kind, tree.ID, tree.version, tree.kind)
			return
		}
	}
	for _, tree := range d.Trees {
		if tree.kind == kind {
			node = &tree
			klog.V(6).Infof("[%s:%s]=>[%s,%s,%s]find node ID:%s \tBy K(%s)", apiVersion, kind, group, version, kind, tree.ID, tree.kind)
			return
		}
	}
	node = d.Fetch(kind)
	klog.V(6).Infof("[%s:%s]=>[%s,%s,%s]find node ID:%s \tBy FetchKind(%s)", apiVersion, kind, group, version, kind, node.ID, node.kind)
	return node
}
