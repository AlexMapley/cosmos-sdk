/*
To prevent namespace collision between consumer modules, we define a type
Subspace. A Subspace can only be generated by the keeper, and the keeper checks
the existence of the Subspace having the same name before generating the
Subspace.

Consumer modules must take a Subspace (via Keeper.Subspace), not the keeper
itself. This isolates each modules from the others and make them modify their
respective parameters safely. Keeper can be treated as master permission for all
Subspaces (via Keeper.GetSubspace), so should be passed to proper modules
(ex. x/governance).
*/
package types