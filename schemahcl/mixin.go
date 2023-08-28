package schemahcl

import (
	"errors"
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (s *State) applyMixin(b *hclsyntax.Block, mixins map[string]*hclsyntax.Block) error {
	if b.Body == nil || b.Body.Attributes == nil {
		return nil
	}

	embedAttribute, ok := b.Body.Attributes[embedAttr]
	if !ok {
		return nil
	}

	mixinsToApply, err := getMixinNames(embedAttribute)
	if err != nil {
		return err
	}

	for _, mixinName := range mixinsToApply {
		mixin, ok := mixins[mixinName]
		if !ok {
			return nil
		}

		if err := mergeBlocks(b, mixin); err != nil {
			return err
		}
		mergeAttributes(b, mixin)
	}

	delete(b.Body.Attributes, embedAttr)
	return nil
}

func getMixinNames(embedAttribute *hclsyntax.Attribute) ([]string, error) {
	if expr, ok := embedAttribute.Expr.(*hclsyntax.ScopeTraversalExpr); ok {
		return []string{expr.Traversal[1].(hcl.TraverseAttr).Name}, nil
	} else if expr, ok := embedAttribute.Expr.(*hclsyntax.TupleConsExpr); ok {
		names := make([]string, 0, len(expr.Exprs))
		for _, expression := range expr.Exprs {
			if expr, ok := expression.(*hclsyntax.ScopeTraversalExpr); ok {
				names = append(names, expr.Traversal[1].(hcl.TraverseAttr).Name)
			}
		}
		return names, nil
	}

	return nil, fmt.Errorf("schemahcl: failed to get mixin names: %w", errors.New("invalid mixin attribute type"))
}

func mergeBlocks(b *hclsyntax.Block, mixin *hclsyntax.Block) error {
	for _, blk := range mixin.Body.Blocks {
		switch blk.Type {
		case "column":
			b.Body.Blocks = append(b.Body.Blocks, blk)
		default:
			err := mergeSingleBlock(b, blk)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func mergeSingleBlock(b *hclsyntax.Block, blk *hclsyntax.Block) error {
	existingBlock := getExistingBlock(b, blk.Type)

	if existingBlock == nil {
		b.Body.Blocks = append(b.Body.Blocks, blk)
		return nil
	}

	return mergeAttributesOfBlock(existingBlock, blk)
}

func getExistingBlock(b *hclsyntax.Block, blkType string) *hclsyntax.Block {
	for _, block := range b.Body.Blocks {
		if block.Type == blkType {
			return block
		}
	}
	return nil
}

func mergeAttributesOfBlock(existingBlock, blk *hclsyntax.Block) error {
	for k, v := range blk.Body.Attributes {
		if _, ok := existingBlock.Body.Attributes[k]; !ok {
			existingBlock.Body.Attributes[k] = v
			continue
		}

		valuesToAdd, err := getValuesToAdd(v)
		if err != nil {
			return err
		}

		if err := mergeAttributeValue(existingBlock, k, valuesToAdd); err != nil {
			return err
		}
	}
	return nil
}

func getValuesToAdd(v *hclsyntax.Attribute) (*hclsyntax.TupleConsExpr, error) {
	if vTuple, ok := v.Expr.(*hclsyntax.TupleConsExpr); ok {
		return vTuple, nil
	} else if vSingle, ok := v.Expr.(*hclsyntax.ScopeTraversalExpr); ok {
		return &hclsyntax.TupleConsExpr{
			Exprs:     []hclsyntax.Expression{vSingle},
			SrcRange:  vSingle.SrcRange,
			OpenRange: vSingle.SrcRange,
		}, nil
	}
	return nil, fmt.Errorf("schemahcl: failed to merge attributes: %w", errors.New("invalid mixin attribute type"))
}

func mergeAttributeValue(existingBlock *hclsyntax.Block, k string, valuesToAdd *hclsyntax.TupleConsExpr) error {
	attr := existingBlock.Body.Attributes[k]
	if attrTuple, ok := attr.Expr.(*hclsyntax.TupleConsExpr); ok {
		attr.Expr.(*hclsyntax.TupleConsExpr).Exprs = append(attrTuple.Exprs, valuesToAdd.Exprs...)
		return nil
	} else if attrSingle, ok := attr.Expr.(*hclsyntax.ScopeTraversalExpr); ok {
		attr.Expr = &hclsyntax.TupleConsExpr{
			Exprs:     []hclsyntax.Expression{attrSingle},
			SrcRange:  attrSingle.SrcRange,
			OpenRange: attrSingle.SrcRange,
		}
		attr.Expr.(*hclsyntax.TupleConsExpr).Exprs = append(attrTuple.Exprs, valuesToAdd.Exprs...)
		return nil
	}
	return fmt.Errorf("schemahcl: failed to merge attributes: %w", errors.New("invalid source attribute type"))
}

func mergeAttributes(b *hclsyntax.Block, mixin *hclsyntax.Block) {
	for k, v := range mixin.Body.Attributes {
		if _, ok := b.Body.Attributes[k]; !ok {
			b.Body.Attributes[k] = v
		}
	}
}
