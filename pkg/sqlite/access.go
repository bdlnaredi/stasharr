package sqlite

// SQL fragments enforcing a user's tier (library access policy) and
// selecting a user's personal history rows.

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"

	"github.com/stashapp/stash/pkg/access"
)

func intList(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// folderPathCondition matches a folder path column against the allowed paths.
func folderPathCondition(col string, paths []string) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	for _, p := range paths {
		p = strings.TrimRight(p, `/\`)
		if p == "" {
			continue
		}
		clauses = append(clauses, fmt.Sprintf(`(%[1]s = ? OR %[1]s LIKE ? ESCAPE '\' OR %[1]s LIKE ? ESCAPE '\')`, col))
		args = append(args, p, escapeLike(p)+"/%", escapeLike(p)+`\%`)
	}
	if len(clauses) == 0 {
		return "0", nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

// policyCondition builds an SQL condition restricting an object id column.
//
//	filesJoin:  table joining object id to file id, eg scenes_files
//	objCol:     the object id column in the join tables, eg scene_id
//	tagsJoin:   table joining object id to tag id, eg scenes_tags
//	idExpr:     the outer expression holding the object id, eg scenes.id
//	extraPath:  optional additional EXISTS clause matching on path (galleries' folder)
func policyCondition(p *access.Policy, filesJoin, objCol, tagsJoin, idExpr, extraPath string) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if len(p.AllowedPaths) > 0 {
		pathCond, pathArgs := folderPathCondition("acc_fo.path", p.AllowedPaths)
		c := fmt.Sprintf(`EXISTS (SELECT 1 FROM %[1]s acc_j JOIN files acc_f ON acc_f.id = acc_j.file_id JOIN folders acc_fo ON acc_fo.id = acc_f.parent_folder_id WHERE acc_j.%[2]s = %[3]s AND %[4]s)`,
			filesJoin, objCol, idExpr, pathCond)
		args = append(args, pathArgs...)
		if extraPath != "" {
			extraCond, extraArgs := folderPathCondition("acc_gf.path", p.AllowedPaths)
			c = "(" + c + " OR " + fmt.Sprintf(extraPath, idExpr, extraCond) + ")"
			args = append(args, extraArgs...)
		}
		conds = append(conds, c)
	}

	if len(p.BlockedTagIDs) > 0 && tagsJoin != "" {
		conds = append(conds, fmt.Sprintf(`NOT EXISTS (SELECT 1 FROM %s acc_t WHERE acc_t.%s = %s AND acc_t.tag_id IN (%s))`,
			tagsJoin, objCol, idExpr, intList(p.BlockedTagIDs)))
	}

	if len(p.RequiredTagIDs) > 0 && tagsJoin != "" {
		conds = append(conds, fmt.Sprintf(`EXISTS (SELECT 1 FROM %s acc_rt WHERE acc_rt.%s = %s AND acc_rt.tag_id IN (%s))`,
			tagsJoin, objCol, idExpr, intList(p.RequiredTagIDs)))
	}

	if len(conds) == 0 {
		return "", nil
	}
	return "(" + strings.Join(conds, " AND ") + ")", args
}

func sceneAccessCondition(ctx context.Context, idExpr string) (string, []interface{}) {
	p := access.ContentPolicy(ctx)
	if p == nil {
		return "", nil
	}
	return policyCondition(p, "scenes_files", "scene_id", "scenes_tags", idExpr, "")
}

// sceneAccessWhere returns " AND <cond>" or "" for raw SQL.
func sceneAccessWhere(ctx context.Context, idExpr string) (string, []interface{}) {
	c, args := sceneAccessCondition(ctx, idExpr)
	if c == "" {
		return "", nil
	}
	return " AND " + c, args
}

func imageAccessCondition(ctx context.Context, idExpr string) (string, []interface{}) {
	p := access.ContentPolicy(ctx)
	if p == nil {
		return "", nil
	}
	return policyCondition(p, "images_files", "image_id", "images_tags", idExpr, "")
}

func galleryAccessCondition(ctx context.Context, idExpr string) (string, []interface{}) {
	p := access.ContentPolicy(ctx)
	if p == nil {
		return "", nil
	}
	// folder based galleries are matched on their folder path
	folderClause := `EXISTS (SELECT 1 FROM galleries acc_g JOIN folders acc_gf ON acc_gf.id = acc_g.folder_id WHERE acc_g.id = %s AND %s)`
	return policyCondition(p, "galleries_files", "gallery_id", "galleries_tags", idExpr, folderClause)
}

func markerAccessCondition(ctx context.Context, sceneIDExpr string, markerIDExpr string) (string, []interface{}) {
	p := access.ContentPolicy(ctx)
	if p == nil {
		return "", nil
	}
	c, args := policyCondition(p, "scenes_files", "scene_id", "scenes_tags", sceneIDExpr, "")
	if len(p.BlockedTagIDs) > 0 {
		blocked := fmt.Sprintf(`NOT EXISTS (SELECT 1 FROM scene_markers acc_m WHERE acc_m.id = %[1]s AND acc_m.primary_tag_id IN (%[2]s)) AND NOT EXISTS (SELECT 1 FROM scene_markers_tags acc_mt WHERE acc_mt.scene_marker_id = %[1]s AND acc_mt.tag_id IN (%[2]s))`,
			markerIDExpr, intList(p.BlockedTagIDs))
		if c == "" {
			c = "(" + blocked + ")"
		} else {
			c = "(" + c + " AND " + blocked + ")"
		}
	}
	return c, args
}

func tagAccessCondition(ctx context.Context, idExpr string) (string, []interface{}) {
	p := access.ContentPolicy(ctx)
	if p == nil || len(p.BlockedTagIDs) == 0 {
		return "", nil
	}
	return fmt.Sprintf("(%s NOT IN (%s))", idExpr, intList(p.BlockedTagIDs)), nil
}

func goquCond(c string, args []interface{}) exp.Expression {
	return goqu.L(c, args...)
}

// withAccess adds an access condition to a goqu dataset.
func withAccess(q *goqu.SelectDataset) func(c string, args []interface{}) *goqu.SelectDataset {
	return func(c string, args []interface{}) *goqu.SelectDataset {
		if c == "" {
			return q
		}
		return q.Where(goquCond(c, args))
	}
}

// addAccessWhere adds an access condition to a query builder.
func addAccessWhere(query *queryBuilder, c string, args []interface{}) {
	if c == "" {
		return
	}
	query.addWhere(c)
	query.addArg(args...)
}

// historyUserCond returns a condition selecting the current user's rows of a
// history table (scenes_view_dates / scenes_o_dates).
func historyUserCond(ctx context.Context, tableAlias string) string {
	uid := access.HistoryUserID(ctx)
	if uid == nil {
		return tableAlias + ".user_id IS NULL"
	}
	return fmt.Sprintf("%s.user_id = %d", tableAlias, *uid)
}

func historyUserExpr(ctx context.Context, col exp.IdentifierExpression) exp.Expression {
	uid := access.HistoryUserID(ctx)
	if uid == nil {
		return col.IsNull()
	}
	return col.Eq(*uid)
}

// sceneRatingExpr returns the expression for the current user's scene rating.
func sceneRatingExpr(ctx context.Context) string {
	uid := access.HistoryUserID(ctx)
	if uid == nil {
		return "scenes.rating"
	}
	return fmt.Sprintf("(SELECT acc_us.rating FROM users_scenes acc_us WHERE acc_us.user_id = %d AND acc_us.scene_id = scenes.id)", *uid)
}
