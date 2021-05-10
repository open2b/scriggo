// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

package ast

// Ctx represents a Show context.
type Ctx byte

const (
	CtxText                    = Ctx(FormatText << 4)
	CtxHTML                    = Ctx(FormatHTML << 4)
	CtxHTMLTag                 = CtxHTML | 0b0100
	CtxHTMLUnquotedAttr        = CtxHTML | 0b1000
	CtxHTMLQuotedAttr          = CtxHTML | 0b1100
	CtxHTMLUnquotedAttrURL     = CtxHTMLUnquotedAttr | 0b0010
	CtxHTMLQuotedAttrURL       = CtxHTMLQuotedAttr | 0b0010
	CtxHTMLUnquotedAttrURLSet  = CtxHTMLUnquotedAttr | 0b0011
	CtxHTMLQuotedAttrURLSet    = CtxHTMLQuotedAttr | 0b0011
	CtxCSS                     = Ctx(FormatCSS << 4)
	CtxCSSString               = CtxCSS | 0b0010
	CtxJS                      = Ctx(FormatJS << 4)
	CtxJSString                = CtxJS | 0b0010
	CtxJSUnquotedAttr          = CtxJS | 0b1000
	CtxJSQuotedAttr            = CtxJS | 0b1100
	CtxJSUnquotedAttrString    = CtxJSUnquotedAttr | 0b0010
	CtxJSQuotedAttrString      = CtxJSUnquotedAttr | 0b0010
	CtxJSON                    = Ctx(FormatJSON << 4)
	CtxJSONString              = CtxJSON | 0b0010
	CtxMarkdown                = Ctx(FormatMarkdown << 4)
	CtxMarkdownURL             = CtxMarkdown | 0b0010
	CtxMarkdownSpacesCodeBlock = CtxMarkdown | 0b1000
	CtxMarkdownTabCodeBlock    = CtxMarkdown | 0b0100
)

func FormatToCtx(format Format) Ctx {
	return Ctx(format << 4)
}

func (ctx Ctx) Format() Format {
	return Format(ctx >> 4)
}

func (ctx Ctx) InTag() bool {
	return Format(ctx) == FormatHTML && ctx&0b1111 == 0b0100
}

func (ctx Ctx) InURL() bool {
	return (Format(ctx) == FormatHTML || Format(ctx) == FormatMarkdown) && ctx&0b0010 != 0
}

func (ctx Ctx) Set() bool {
	return Format(ctx) == FormatHTML && ctx&0b0001 != 0
}

func (ctx Ctx) InAttribute() bool {
	return (Format(ctx) == FormatHTML || Format(ctx) == FormatJS) && ctx&0b1000 != 0
}

func (ctx Ctx) Quoted() bool {
	return (Format(ctx) == FormatHTML || Format(ctx) == FormatJS) && ctx&0b1100 != 0
}

func (ctx Ctx) InString() bool {
	return (Format(ctx) == FormatJS || Format(ctx) == FormatCSS) && ctx&0b0010 != 0
}

func (ctx Ctx) InSpacesCodeBlock() bool {
	return Format(ctx) == FormatMarkdown && ctx&0b1000 != 0
}

func (ctx Ctx) InTabCodeBlock() bool {
	return Format(ctx) == FormatMarkdown && ctx&0b0100 != 0
}

func (ctx Ctx) String() string {
	s := ctx.Format().String()
	return s
}

func MakeHTMLURLCtx(quoted, set bool) Ctx {
	if quoted {
		if set {
			return CtxHTMLQuotedAttrURLSet
		}
		return CtxHTMLQuotedAttrURL
	}
	if set {
		return CtxHTMLUnquotedAttrURLSet
	}
	return CtxHTMLUnquotedAttrURL
}
