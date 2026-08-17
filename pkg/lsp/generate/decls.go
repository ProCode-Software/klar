package main

import "github.com/ProCode-Software/klar/pkg/lsp/rpc"

// Common to all meta model objects
type BaseDecl struct {
	Name string `json:"name"`
	/**
	 * An optional documentation;
	 */
	Documentation string `json:"documentation,omitempty"`
	/**
	 * Since when (release number) this declaration is
	 * available. Is undefined if not known.
	 */
	Since string `json:"since,omitempty"`
	/**
	 * All since tags in case there was more than one tag.
	 * Is undefined if not known.
	 */
	SinceTags []string `json:"sinceTags,omitempty"`
	/**
	 * Whether this is a proposed declaration. If omitted,
	 * the declaration is final.
	 */
	Proposed bool `json:"proposed,omitempty"`
	/**
	 * Whether the declaration is deprecated or not. If deprecated
	 * the property contains the deprecation message.
	 */
	Deprecated string `json:"deprecated,omitempty"`
}

type Request struct {
	BaseDecl
	Method   string                    `json:"method"`
	TypeName string                    `json:"typeName,omitempty"`
	Params   *rpc.Union2[Type, []Type] `json:"params,omitempty"`
	Result   *Type                     `json:"result"`
	/**
	 * Optional partial result type if the request
	 * supports partial result reporting.
	 */
	PartialResult *Type `json:"partialResult,omitempty"`

	/**
	 * An optional error data type.
	 */
	ErrorData *Type `json:"errorData,omitempty"`

	/**
	 * Optional a dynamic registration method if it
	 * different from the request's method.
	 */
	RegistrationMethod string `json:"registrationMethod,omitempty"`

	/**
	 * Optional registration options if the request
	 * supports dynamic registration.
	 */
	RegistrationOptions *Type `json:"registrationOptions,omitempty"`

	/**
	 * The direction in which this request is sent
	 * in the protocol.
	 */
	MessageDirection MessageDirection `json:"messageDirection"`

	/**
	 * The client capability property path if any.
	 */
	ClientCapability string `json:"clientCapability,omitempty"`

	/**
	 * The server capability property path if any.
	 */
	ServerCapability string `json:"serverCapability,omitempty"`
}

type Notification struct {
	BaseDecl
	/**
	 * The notifications's method name.
	 */
	Method string `json:"method"`

	/**
	 * The type name of the notifications if any.
	 */
	TypeName string `json:"typeName,omitempty"`

	/**
	 * The parameter type(s) if any.
	 */
	Params *rpc.Union2[Type, []Type] `json:"params,omitempty"`

	/**
	 * Optional a dynamic registration method if it
	 * different from the notifications's method.
	 */
	RegistrationMethod string `json:"registrationMethod,omitempty"`

	/**
	 * Optional registration options if the notification
	 * supports dynamic registration.
	 */
	RegistrationOptions *Type `json:"registrationOptions,omitempty"`

	/**
	 * The direction in which this notification is sent
	 * in the protocol.
	 */
	MessageDirection MessageDirection `json:"messageDirection"`

	/**
	 * The client capability property path if any.
	 */
	ClientCapability string `json:"clientCapability,omitempty"`

	/**
	 * The server capability property path if any.
	 */
	ServerCapability string `json:"serverCapability,omitempty"`
}

type Structure struct {
	BaseDecl
	/**
	 * Structures extended from. This structures form
	 * a polymorphic type hierarchy.
	 */
	Extends []Type `json:"extends,omitempty"`

	/**
	 * Structures to mix in. The properties of these
	 * structures are `copied` into this structure.
	 * Mixins don't form a polymorphic type hierarchy in
	 * LSP.
	 */
	Mixins []Type `json:"mixins,omitempty"`

	/**
	 * The properties.
	 */
	Properties []Property `json:"properties"`
}

type Enumeration struct {
	BaseDecl
	/**
	 * The type of the elements.
	 */
	Type BaseType // Representing string | integer | uinteger

	/**
	 * The enum values.
	 */
	Values []EnumerationEntry `json:"values"`

	/**
	 * Whether the enumeration supports custom values (e.g. values which are not
	 * part of the set defined in `values`). If omitted no custom values are
	 * supported.
	 */
	SupportsCustomValues bool `json:"supportsCustomValues,omitempty"`
}

type TypeAlias struct {
	BaseDecl
	/**
	 * The aliased type.
	 */
	Type Type `json:"type"`
}

type Property struct {
	BaseDecl
	/**
	 * The type of the property
	 */
	Type Type `json:"type"`
	/**
	 * Whether the property is optional. If
	 * omitted, the property is mandatory.
	 */
	Optional bool `json:"optional,omitempty"`
}

type EnumerationEntry struct {
	BaseDecl
	// The value.
	Value any `json:"value"` // string | number
}
