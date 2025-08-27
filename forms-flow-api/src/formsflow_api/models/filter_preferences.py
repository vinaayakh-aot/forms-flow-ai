"""This manages filter preference Database Models."""

from __future__ import annotations

from typing import List

from formsflow_api_utils.utils.enums import FilterStatus
from sqlalchemy import Index, UniqueConstraint, and_, or_, tuple_
from sqlalchemy.orm import relationship

from .audit_mixin import AuditDateTimeMixin
from .base_model import BaseModel
from .db import db
from .filter import Filter


class FilterPreferences(db.Model, BaseModel, AuditDateTimeMixin):
    """Stores user-specific preferences for filters, including sort order and visibility."""

    __tablename__ = "filter_preferences"
    id = db.Column(
        db.Integer,
        primary_key=True,
        comment="Primary key for the User Preference entry.",
    )
    tenant = db.Column(
        db.String,
        nullable=True,
        comment="Tenant identifier (optional, for multi-tenant support).",
    )
    user_id = db.Column(
        db.String, index=True, nullable=False, comment="Unique identifier for the user."
    )
    filter_id = db.Column(
        db.Integer,
        db.ForeignKey("filter.id", ondelete="CASCADE"),
        comment="Reference to the filter ID.",
    )
    sort_order = db.Column(
        db.Integer, comment="Sort order preference for the applied filter."
    )
    hide = db.Column(
        db.Boolean,
        default=False,
        nullable=False,
        comment="Flag to indicate if the filter is hidden.",
    )

    filter = relationship("Filter", lazy="noload", backref="filter_preferences")

    __table_args__ = (
        UniqueConstraint("user_id", "filter_id", name="_user_filter_uc"),
        Index("idx_user_id_and_tenant", "tenant", "user_id"),
    )

    @classmethod
    def bulk_upsert_preferences(cls, preferences_list: List[dict], tenant_key: str):
        """Upsert in filter preferences."""
        if not preferences_list:
            return []

        # exctract userid and filter id for search
        keys = [(p["user_id"], p["filter_id"]) for p in preferences_list]
        # fetch existing data
        query = cls.query.filter(tuple_(cls.user_id, cls.filter_id).in_(keys))
        if tenant_key:
            query.filter(cls.tenant == tenant_key)
        # feth all existing records
        existing_records = query.all()
        # create a existing data lookup dict
        existing_lookup = {
            (r.user_id, r.filter_id, r.tenant): r for r in existing_records
        }
        for pref in preferences_list:
            key = (pref["user_id"], pref["filter_id"], pref.get("tenant"))
            if key in existing_lookup:
                # Update existing
                record = existing_lookup[key]
                record.sort_order = pref.get("sort_order")
                record.hide = pref.get("hide", False)
            else:
                # Create new
                new_record = cls(
                    user_id=pref["user_id"],
                    filter_id=pref["filter_id"],
                    sort_order=pref.get("sort_order"),
                    tenant=pref.get("tenant"),
                    hide=pref.get("hide", False),
                )
                db.session.add(new_record)
        return db.session.commit()

    @classmethod
    def get_filters_by_user_id(
        cls,
        user_id: str,
        tenant: str,
        filter_type: str,
        roles: List[str],
        parent_filter_id: int = None,
    ) -> List[FilterPreferences]:
        """
        Get user's filter preferences with authorization checks.
        
        This method performs a complex query that:
        1. Finds user's saved filter preferences
        2. Joins with the filter table to get filter details
        3. Applies authorization rules (user can only see their own filters + public filters)
        4. Uses eager loading to avoid N+1 queries
        
        :param user_id: Username to get preferences for
        :param tenant: Tenant key for multi-tenancy
        :param filter_type: Type of filters (TASK, ATTRIBUTE)
        :param roles: User's roles for authorization (currently unused but kept for API compatibility)
        :param parent_filter_id: Parent filter ID for nested filters
        :return: List of FilterPreferences with populated filter relationships
        """
        from sqlalchemy.orm import aliased, contains_eager
        from sqlalchemy import or_, and_
        
        # Build base query for user's filter preferences
        base_query = cls.query.filter(cls.user_id == user_id)
        if tenant:
            base_query = base_query.filter(cls.tenant == tenant)

        # Join with filter table to get filter details and apply constraints
        filter_alias = aliased(Filter)
        query = base_query.join(filter_alias, cls.filter_id == filter_alias.id)

        # Apply filter constraints (uses database indexes for performance)
        query = query.filter(
            filter_alias.status == FilterStatus.ACTIVE.value,
            filter_alias.filter_type == filter_type,
        )
        if parent_filter_id:
            query = query.filter(filter_alias.parent_filter_id == parent_filter_id)

        # Use eager loading to populate filter relationship in single query
        # This prevents N+1 queries when accessing preference.filter later
        query = query.options(contains_eager(cls.filter, alias=filter_alias))
        
        # Apply authorization rules: user can see their own filters + public filters
        query = query.filter(
            or_(
                # User's own filters (they created them)
                filter_alias.created_by == user_id,
                # Public filters (no role/user restrictions)
                and_(
                    or_(filter_alias.roles.is_(None), filter_alias.roles == []),
                    or_(filter_alias.users.is_(None), filter_alias.users == []),
                    filter_alias.created_by != user_id  # Not user's own filter
                )
            )
        )
        
        # Order by user's custom sort preference
        query = query.order_by(cls.sort_order.asc())
        
        return query.all()
