import { CriterionModifier, FilterMode } from "src/core/generated-graphql";
import { ListFilterModel } from "src/models/list-filter/filter";
import { SceneListFilterOptions } from "src/models/list-filter/scenes";
import {
  Criterion,
  ModifierCriterion,
} from "src/models/list-filter/criteria/criterion";

// Link to the Scenes list showing only audio files: no video codec, but an
// audio codec.
export function audioScenesLink(): string {
  const filter = new ListFilterModel(FilterMode.Scenes);
  const criteria: Criterion[] = [];
  for (const [type, modifier] of [
    ["video_codec", CriterionModifier.IsNull],
    ["audio_codec", CriterionModifier.NotNull],
  ] as const) {
    const option = SceneListFilterOptions.criterionOptions.find(
      (o) => o.type === type
    );
    if (!option) continue;
    const c = option.makeCriterion();
    if (c instanceof ModifierCriterion) {
      c.modifier = modifier;
    }
    criteria.push(c);
  }
  return `/scenes?${filter.setCriteria(criteria).makeQueryParameters()}`;
}
