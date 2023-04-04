package options

import (
	"errors"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/utils"
)

func (opt *NodeOptions) NewChainJWT(name string, exp time.Time) (string, error) {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	if opt.JWT.Chain == nil {
		opt.JWT.Chain = &JWTListItem{}
	}

	if opt.JWT.Chain.Normal == nil {
		opt.JWT.Chain.Normal = []*TokenItem{}
	}

	token, err := utils.NewJWTToken(name, "chain", "*", opt.JWT.Key, exp)
	if err != nil {
		return "", err
	}
	opt.JWT.Chain.Normal = append(opt.JWT.Chain.Normal, &TokenItem{Remark: name, Token: token})

	return token, opt.writeToconfig()
}

func (opt *NodeOptions) GetOrCreateNodeJwt(groupid, name string, exp time.Time) (string, error) {
	if opt.JWT.Node == nil {
		opt.JWT.Node = map[string]*JWTListItem{}
	}

	g, ok := opt.JWT.Node[groupid]
	if !ok {
		opt.JWT.Node[groupid] = &JWTListItem{}
		g = opt.JWT.Node[groupid]
	}

	var token string
	for _, v := range g.Normal {
		if v.Remark == name {
			token = v.Token
			break
		}
	}

	if token != "" {
		return token, nil
	}

	return opt.NewNodeJWT(groupid, name, exp)
}

func (opt *NodeOptions) NewNodeJWT(groupid, name string, exp time.Time) (string, error) {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	if opt.JWT.Node == nil {
		opt.JWT.Node = map[string]*JWTListItem{}
	}

	g, ok := opt.JWT.Node[groupid]
	if !ok {
		opt.JWT.Node[groupid] = &JWTListItem{}
		g = opt.JWT.Node[groupid]
	}

	token, err := utils.NewJWTToken(name, "node", groupid, opt.JWT.Key, exp)
	if err != nil {
		return "", err
	}
	g.Normal = append(g.Normal, &TokenItem{Remark: name, Token: token})

	opt.JWT.Node[groupid] = g
	return token, opt.writeToconfig()
}

func (opt *NodeOptions) RevokeChainJWT(token string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	var tokens []*TokenItem
	for _, v := range opt.JWT.Chain.Normal {
		if v.Token == token {
			opt.JWT.Chain.Revoke = append(opt.JWT.Chain.Revoke, v)
			continue
		}
		tokens = append(tokens, v)
	}
	opt.JWT.Chain.Normal = tokens

	return opt.writeToconfig()
}

func (opt *NodeOptions) RevokeNodeJWT(groupid, token string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	g, ok := opt.JWT.Node[groupid]
	if !ok {
		return errors.New("not find jwt for this group")
	}

	var tokens []*TokenItem
	for _, v := range g.Normal {
		if v.Token == token {
			g.Revoke = append(g.Revoke, v)
			continue
		}
		tokens = append(tokens, v)
	}
	g.Normal = tokens

	return opt.writeToconfig()
}

func (opt *NodeOptions) RemoveChainJWT(token string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	tokens := []*TokenItem{}
	for _, v := range opt.JWT.Chain.Normal {
		if v.Token == token {
			continue
		}
		tokens = append(tokens, v)
	}
	opt.JWT.Chain.Normal = tokens

	// remove from revoke
	tokens = []*TokenItem{}
	for _, v := range opt.JWT.Chain.Revoke {
		if v.Token == token {
			continue
		}
		tokens = append(tokens, v)
	}
	opt.JWT.Chain.Revoke = tokens

	return opt.writeToconfig()
}

func (opt *NodeOptions) RemoveNodeJWT(groupid, token string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()

	g, ok := opt.JWT.Node[groupid]
	if !ok {
		return errors.New("not find jwt")
	}

	tokens := []*TokenItem{}
	for _, v := range g.Normal {
		if v.Token == token {
			continue
		}
		tokens = append(tokens, v)
	}
	g.Normal = tokens

	tokens = []*TokenItem{}
	for _, v := range g.Revoke {
		if v.Token == token {
			continue
		}
		tokens = append(tokens, v)
	}
	g.Revoke = tokens

	return opt.writeToconfig()
}

func (opt *NodeOptions) GetAllJWT() (*JWT, error) {
	return opt.JWT, nil
}

func (opt *NodeOptions) IsValidChainJWT(token string) bool {
	ok, err := utils.IsJWTTokenValid(token, opt.JWT.Key)
	if err != nil {
		logger.Warnf("jwt is not valid: %s", err)
		return false
	}
	if !ok {
		return false
	}

	for _, v := range opt.JWT.Chain.Normal {
		if token == v.Token {
			return true
		}
	}

	return false
}

func (opt *NodeOptions) IsValidNodeJWT(groupid, token string) bool {
	g, ok := opt.JWT.Node[groupid]
	if !ok {
		return false
	}

	ok, err := utils.IsJWTTokenValid(token, opt.JWT.Key)
	if err != nil {
		return false
	}
	if !ok {
		return false
	}

	for _, v := range g.Normal {
		if token == v.Token {
			return true
		}
	}

	return false
}
